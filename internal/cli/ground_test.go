package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/action/claude"
	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/shim"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/wiring"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// A worktree that opens on a command replaces the process, so a case that
	// watches one types it in a child of this binary.
	if len(os.Args) > 1 && os.Args[1] == underTest {
		logLevel := LogLevel()
		slog.SetDefault(logger(os.Stderr, logLevel))
		os.Exit(Execute(stubVersion, wiring.Wire, os.Args[2:], os.Stdout, logLevel))
	}
	testenv.Main(m)
}

// stubVersion is what the tree is built with, which --version alone reads.
const stubVersion = "v0.0.0-test"

// versionLine is the whole of what --version prints: the test binary and the
// tree it builds are compiled by the one toolchain.
var versionLine = "work version " + stubVersion + " (" + runtime.Version() + ")\n"

// defaultDir is where worktrees go with nothing configured.
var defaultDir = config.Default().Worktree.Dir()

// underTest is the first word a child of the test binary runs work under. It
// sits in the argument list rather than the environment, which whatever the
// handoff execs would inherit.
const underTest = "--work-under-test"

// session is one shell standing in a repository of the case's own: git runs for
// real, every other tool stands in, and a command is typed as the user types it.
type session struct {
	t *testing.T

	// Repo is the main checkout, its symlinks resolved: git reports a worktree
	// resolved, and what work prints of one is git's own answer.
	Repo string

	// Origin is the repository the session's origin points at, empty where it was
	// given none. A case reads what the forge was asked against it.
	Origin string

	// Dir is where the shell stands, which is what standing in a worktree is
	// judged against.
	Dir string

	// Shim is the file the shell function named for the worktree, which a command
	// answering with one writes rather than prints. Emptied, the session is a shell
	// that never sourced the integration.
	Shim string

	// asked is what the stand-ins were asked to run, one line per invocation, in
	// the order they ran.
	asked func() []string

	// seen is how many of those the runs before this one accounted for.
	seen int
}

// resolved is a path with its symlinks taken out, which is how git reports a
// worktree and so how work prints one.
func resolved(t *testing.T, path string) string {
	t.Helper()
	out, err := filepath.EvalSymlinks(path)
	require.NoError(t, err, "resolve %s", path)
	return out
}

// repository stands a shell in a fresh repository holding one empty commit on
// main. A stub given here stands in for the tool of that name in place of the
// one that only fails.
func repository(t *testing.T, answering ...testenv.Stub) *session {
	t.Helper()
	repo := resolved(t, testenv.InitRepo(t))
	return &session{
		t:     t,
		Repo:  repo,
		Dir:   repo,
		Shim:  filepath.Join(t.TempDir(), "answer"),
		asked: testenv.Stubs(t, slices.Concat(testenv.Tools(), answering)...),
	}
}

// result is what one invocation came to: the worktree it answered the shell
// with, what it printed, what it said at each level, what the stand-ins were
// asked, and the status it exited on. Answered is what the run wrote into the
// file the shell named and Out what a shell that named none reads instead, empty
// where the case named a reader of its own. [result.records] is what was said
// whole. A case states the whole of one with [result.came].
type result struct {
	Code     int
	Answered string
	Out      string
	Errored  []string
	Warned   []string
	Informed []string
	Debugged []string
	Asked    []string

	records []record
}

// apart leaves what a run said out of the whole [result.came] compares.
var apart = besides("Errored", "Warned", "Informed", "Debugged")

// came fails the case unless the run came to want in every part, go-cmp's diff
// of the two being the whole message: what the case promises is the comment
// above it. What a case leaves unnamed is what the run should not have
// produced: nothing printed, nothing said, no tool asked. A field a case states
// on its own is left out of the comparison with [besides].
func (r result) came(t *testing.T, want result, opts ...cmp.Option) {
	t.Helper()
	testenv.Equal(t, want, r, "what the run came to", append(opts, rendered)...)
}

// rendered leaves out the records behind what a run said, which [result.under]
// reads and no case compares.
var rendered = cmpopts.IgnoreUnexported(result{})

// besides names the fields a case states on their own, which [result.came] then
// leaves out of the whole it compares.
func besides(fields ...string) cmp.Option { return cmpopts.IgnoreFields(result{}, fields...) }

// atOnce compares what the stand-ins were asked as a set: the listings one run
// gathers are asked for at once and settle no order between them. The order a
// run puts its questions in is a case of its own, asserted without this.
var atOnce = cmp.FilterPath(
	func(p cmp.Path) bool { return p.Last().String() == ".Asked" },
	cmpopts.SortSlices(func(a, b string) bool { return a < b }),
)

// refused fails the case unless the run came to nothing but one refusal, whose
// line carries each of these. It is for a refusal spelled somewhere other than
// work; one of work's own is named whole with [result.came].
func (r result) refused(t *testing.T, names string, alongside ...string) {
	t.Helper()
	r.came(t, result{Code: 1}, besides("Errored"))
	r.saying(t, append([]string{names}, alongside...)...)
}

// saying fails the case unless the run refused once, naming each of these, which
// is what a case states beside the rest of a run it names with [besides].
func (r result) saying(t *testing.T, names ...string) {
	t.Helper()
	require.Len(t, r.Errored, 1, "work said %q; want the one refusal", r.Errored)
	for _, want := range names {
		require.Contains(t, r.Errored[0], want, "the refusal does not name %s", want)
	}
}

// run types one work command in this session and hands back what came of it. One
// whose worktree opens on a command replaces the process, and is typed through
// [session.hands] instead.
func (s *session) run(args ...string) result { return s.reads(nil, args...) }

// reads is the same with the reader a case names standing where the shell reads:
// a terminal there is a person reading the path, and what the run prints goes to
// it rather than onto the result.
func (s *session) reads(out io.Writer, args ...string) result {
	t := s.t
	t.Helper()
	t.Chdir(s.Dir)
	s.names()

	var printed bytes.Buffer
	if out == nil {
		out = &printed
	}
	// The process log for the length of this run.
	logLevel := LogLevel()
	var stream bytes.Buffer
	was := slog.Default()
	slog.SetDefault(logger(&stream, logLevel))
	defer slog.SetDefault(was)

	code := Execute(stubVersion, wiring.Wire, args, out, logLevel)

	return s.outcome(code, printed.String(), decoded(t, stream.String()))
}

// outcome is what a run came to, its diagnostics split by level.
func (s *session) outcome(code int, out string, records []record) result {
	return result{Code: code, Answered: s.answered(), Out: out,
		Errored: messagesAt(records, slog.LevelError), Warned: messagesAt(records, slog.LevelWarn),
		Informed: messagesAt(records, slog.LevelInfo), Debugged: messagesAt(records, slog.LevelDebug),
		Asked: s.since(), records: records}
}

// messagesAt is the sentences a run said at one level, in the order it said them.
func messagesAt(records []record, want slog.Level) []string {
	var out []string
	for _, one := range records {
		if one.Level == want {
			out = append(out, one.Message)
		}
	}
	return out
}

// record is one thing a run said: the level, the sentence, and the values behind
// it, keyed as work keyed them.
type record struct {
	Level   slog.Level
	Message string
	Values  map[string]string
}

// logger is the log a run writes, whichever way the command was typed, and so
// what [decoded] reads back.
func logger(w io.Writer, from slog.Leveler) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: from}))
}

// decoded is what a run said, off the stream it wrote its log to.
func decoded(t *testing.T, stream string) []record {
	t.Helper()
	var records []record
	for raw := range strings.SplitSeq(strings.TrimSpace(stream), "\n") {
		if raw == "" {
			continue
		}
		var said struct {
			Level   slog.Level `json:"level"`
			Message string     `json:"msg"`
		}
		var fields map[string]json.RawMessage
		line := []byte(raw)
		if err := errors.Join(json.Unmarshal(line, &said), json.Unmarshal(line, &fields)); err != nil {
			t.Fatalf("read what a run said: %v", err)
		}
		for _, own := range []string{slog.TimeKey, slog.LevelKey, slog.MessageKey} {
			delete(fields, own)
		}
		records = append(records, record{Level: said.Level, Message: said.Message, Values: keyed(fields)})
	}
	return records
}

// keyed is the values one record carried, each spelled as work keyed it: a
// string without the quotes the log wrote it in, anything else as it stands.
func keyed(fields map[string]json.RawMessage) map[string]string {
	out := make(map[string]string, len(fields))
	for key, raw := range fields {
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			text = string(raw)
		}
		out[key] = text
	}
	return out
}

// names puts the file the shell function named in the environment and takes what
// an earlier command wrote into it away, so what is read back is this one's own
// answer.
func (s *session) names() {
	t := s.t
	t.Helper()
	t.Setenv(shim.CDFile, s.Shim)
	if err := os.Remove(s.Shim); err != nil && !os.IsNotExist(err) {
		t.Fatalf("empty %s: %v", s.Shim, err)
	}
}

// since is what the stand-ins were asked by the run just made, which is what
// the session was asked less what the runs before it were.
func (s *session) since() []string {
	was := s.seen
	all := s.asked()
	s.seen = len(all)
	return all[was:]
}

// answered is the worktree the run wrote into the file the shell named, and
// nothing where it answered with none.
func (s *session) answered() string {
	t := s.t
	t.Helper()
	dir, err := os.ReadFile(s.Shim)
	if os.IsNotExist(err) {
		return ""
	}
	if err != nil {
		t.Fatalf("read %s: %v", s.Shim, err)
	}
	return strings.TrimSuffix(string(dir), "\n")
}

// hands types one work command in a child of the test binary. A worktree that
// opens on a command replaces the process work runs in, and only a child can be
// watched doing it.
func (s *session) hands(args ...string) result {
	t := s.t
	t.Helper()
	s.names()
	// Resolved rather than taken from os.Args[0], which a child given a directory of
	// its own would read a relative path against.
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("find the test binary: %v", err)
	}
	child := exec.Command(binary, append([]string{underTest}, args...)...)
	child.Dir = s.Dir
	child.Env = os.Environ()
	var out, stream bytes.Buffer
	child.Stdout, child.Stderr = &out, &stream

	code := 0
	if err := child.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("work %s: %v", strings.Join(args, " "), err)
		}
		code = exit.ExitCode()
	}
	return s.outcome(code, out.String(), decoded(t, stream.String()))
}

// ticket is one bead as bd answers a listing with it.
type ticket struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Type     string `json:"issue_type"`
	Criteria string `json:"acceptance_criteria"`
}

// doable is a ticket every rule of the vetting lets through, which is the one a
// case names where what the ticket says is not what it establishes.
var doable = ticket{ID: "bd-1", Title: "Do a thing", Status: "open", Type: "task", Criteria: "it works"}

// with is a ticket answering otherwise in one of its fields.
func with(t ticket, f func(*ticket)) ticket {
	f(&t)
	return t
}

// tickets renders what bd answers a listing query with.
func tickets(list ...ticket) string {
	out, err := json.Marshal(list)
	if err != nil {
		panic(err)
	}
	return string(out)
}

// trackerOn, forgeOn, miseOn and claudeOn are the settings keys that put the
// tracker, the forge, the runner and the agent in force.
const (
	trackerOn = "[beads]\nenabled = true\n"
	forgeOn   = "[github]\nenabled = true\n"
	miseOn    = "[mise]\nenabled = true\n"
	claudeOn  = "[claude]\nenabled = true\n"
)

// putUp is the picker as a stand-in records being put up, which is one screen
// however many rows it carries. The prompt carries a trailing space of its own.
const putUp = "fzf --height 40% --reverse --prompt " + prompt + " --ansi --delimiter \t --with-nth 2.."

// listed and vetted are the two questions work puts the tracker before it
// answers for an identifier: what there is, and which of it is unblocked.
const (
	listed = "bd list --all --limit 0 --json"
	vetted = "bd ready --limit 0 --json"
)

// creates and claims are what work asks the tracker for a ticket's worktree
// coming into being, and worked every question a ticket worked from scratch is
// put in the order they are put.
func creates(path, branch string) string { return "bd worktree create " + path + " --branch " + branch }
func claims(id string) string            { return "bd update " + id + " --claim" }

func worked(id, path, branch string) []string {
	return []string{listed, vetted, creates(path, branch), claims(id)}
}

// hosted is the origin a case gives its repository where the forge is to read
// one off it and no case fetches from it.
const hosted = "https://github.com/o/r"

// pullRequests is what work asks the forge for the rows it puts up.
func pullRequests(origin string) string {
	return "gh pr list --repo " + origin + " --state open --limit 200 --json number,title"
}

// tracker is a bd that answers the full listing with all and the readiness one
// with ready, and makes the worktree work asks it for.
func tracker(all, ready string) testenv.Stub {
	return testenv.Stub{Name: "bd", Replies: []testenv.Reply{
		{To: []string{"list"}, Says: all},
		{To: []string{"ready"}, Says: ready},
		{To: []string{"worktree", "create"}, Shell: `git worktree add -q "$3" -b "$5"`},
		{To: []string{"update"}},
	}}
}

// tracking stands a shell in a repository whose tracker lists all, calls each of
// ready unblocked, and makes the worktree work asks it for. body is written
// beside the key that switches the tracker on, so a case asking for a system of
// its own keeps the tracker.
func tracking(t *testing.T, all, ready []ticket, body string, answering ...testenv.Stub) *session {
	t.Helper()
	s := repository(t, append([]testenv.Stub{tracker(tickets(all...), tickets(ready...))}, answering...)...)
	s.settings(trackerOn + body)
	return s
}

// reviewHead is the commit the origin in [reviewing] holds for pull request 7,
// which is what a worktree that fetched it stands on.
const reviewHead = "the pull request's head"

// reviewing stands a shell in a repository whose forge is on and whose origin
// holds the head of pull request 7, which is what its worktree checks out. body
// is written beside the key that switches the forge on, so a case asking for a
// system of its own keeps the forge.
func reviewing(t *testing.T, body string, answering ...testenv.Stub) *session {
	t.Helper()
	s := repository(t, answering...)
	s.Origin = testenv.InitRepo(t)
	testenv.Git(t, s.Origin, "commit", "--allow-empty", "-m", reviewHead)
	testenv.Git(t, s.Origin, "update-ref", "refs/pull/7/head", "HEAD")
	testenv.Git(t, s.Repo, "remote", "add", "origin", s.Origin)
	s.settings(forgeOn + body)
	return s
}

// at is where a worktree of that name lands, which is what a case reads what a
// run answered against.
func (s *session) at(name string) string { return filepath.Join(s.Repo, defaultDir, name) }

// opened puts a worktree in the session's repository, on a branch spelled as its
// name.
func (s *session) opened(name string) string { return s.openedOn(name, name) }

// openedOn is the same with the branch spelled otherwise, which is what a system
// naming its own branches leaves behind.
func (s *session) openedOn(dir, branch string) string {
	s.t.Helper()
	path := s.at(dir)
	testenv.Git(s.t, s.Repo, "worktree", "add", "-b", branch, path)
	return path
}

// detached is the same with no branch, which is what a worktree that goes by its
// directory rather than by a branch stands on.
func (s *session) detached(name string) string {
	s.t.Helper()
	path := s.at(name)
	testenv.Git(s.t, s.Repo, "worktree", "add", "--detach", path)
	return path
}

// transcripts is where Claude Code files a directory's conversations under the
// home the test process was given, which is what work reads them back out of.
func (s *session) transcripts(dir string) string {
	return claude.Bucket(testenv.UserHome(), dir)
}

// carries records these conversations for a worktree, which is what work reads
// to decide what one already there opens on.
func (s *session) carries(dir string, ids ...string) {
	s.t.Helper()
	store := s.transcripts(dir)
	for _, id := range ids {
		testenv.Write(s.t, filepath.Join(store, id+".jsonl"), `{"type":"user"}`+"\n")
	}
}

// settings is what the commands after it are read under, and where that landed.
// Each call takes a settings home of its own, so a path an earlier call handed
// back is no longer the file in force.
func (s *session) settings(body string) string {
	s.t.Helper()
	return testenv.Settings(s.t, body)
}

// dirty puts one of each sort of change in the checkout the shell stands in:
// staged, unstaged, untracked and ignored. The worktree directory is ignored, as
// a repository does.
func (s *session) dirty() {
	s.t.Helper()
	testenv.Write(s.t, filepath.Join(s.Dir, "tracked"), "as committed")
	testenv.Write(s.t, filepath.Join(s.Dir, ".gitignore"), "ignored\n"+defaultDir+"/\n")
	testenv.Git(s.t, s.Dir, "add", "tracked", ".gitignore")
	testenv.Git(s.t, s.Dir, "commit", "-m", "a file to change")

	testenv.Write(s.t, filepath.Join(s.Dir, "staged"), "staged")
	testenv.Git(s.t, s.Dir, "add", "staged")
	testenv.Write(s.t, filepath.Join(s.Dir, "tracked"), "changed")
	testenv.Write(s.t, filepath.Join(s.Dir, "untracked"), "untracked")
	testenv.Write(s.t, filepath.Join(s.Dir, "ignored"), "ignored")
}

// hasBranch reports whether the repository still has that branch.
func (s *session) hasBranch(name string) bool {
	s.t.Helper()
	return git.HasBranch(s.Repo, name)
}

// hasWorktree reports whether git still reports a worktree at path.
func (s *session) hasWorktree(path string) bool {
	s.t.Helper()
	list, err := git.Worktrees(s.Repo)
	if err != nil {
		s.t.Fatalf("read the repository's worktrees: %v", err)
	}
	return slices.ContainsFunc(list, func(w git.Worktree) bool { return git.SameDir(w.Path, path) })
}

// screen is the picker as a case reads it: an fzf that records the rows it was
// put up with and dismisses them, so a run ends cancelled with the listing as
// all that came of it.
type screen struct {
	t    *testing.T
	file string
}

// putsUp is a screen recording into a file of its own.
func putsUp(t *testing.T) *screen {
	t.Helper()
	return &screen{t: t, file: filepath.Join(t.TempDir(), "screen")}
}

// stub is the stand-in a session is handed to put its screens up through.
func (s *screen) stub() testenv.Stub {
	return testenv.Stub{Name: "fzf", Shell: "cat > " + s.file, Exits: 1}
}

// answers is the same handing that row back, for a case whose verb goes on to
// act on what was chosen rather than dismissing the screen.
func (s *screen) answers(row string) testenv.Stub {
	put := s.stub()
	put.Says, put.Exits = row, 0
	return put
}

// rows is what the last screen was put up with, taken away as it is read so
// that no run reads the screen before it: the index each row is keyed on is the
// picker's own and not part of the row.
func (s *screen) rows() []string {
	t := s.t
	t.Helper()
	out, err := os.ReadFile(s.file)
	require.NoError(t, err, "the picker was never put up")
	require.NoError(t, os.Remove(s.file), "the screen could not be cleared")
	var got []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			_, row, _ := strings.Cut(line, "\t")
			got = append(got, row)
		}
	}
	return got
}

// plain is a row with the highlight a screen draws an open one in taken off.
func plain(row string) string { return strings.NewReplacer(highlight, "", reset, "").Replace(row) }

// retyped is what the rows a screen was put up with are retyped as: the mark and
// the icon ahead of a name are the screen's own, and the title behind it the
// row's.
func retyped(rows []string) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		name, _, _ := strings.Cut(plain(row), "  ·  ")
		fields := strings.Fields(name)
		out[i] = fields[len(fields)-1]
	}
	return out
}
