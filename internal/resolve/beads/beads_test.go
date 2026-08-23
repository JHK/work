package beads

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/beads"
	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// pattern is the compiled-in branch naming, which a Branch that never reached
// Load answers with.
var pattern config.Branch

var (
	workable = beads.Bead{ID: "one", Title: "A ticket", Status: "open", Type: "task", AcceptanceCriteria: "It works"}
	epic     = with(workable, func(b *beads.Bead) { b.Type = "epic" })
)

var vettings = []struct {
	name  string
	bead  beads.Bead
	ready bool
	want  string // a fragment of the refusal, or "" for workable
}{
	{"workable and ready", workable, true, ""},
	{"in progress needs no ready check", with(workable, func(b *beads.Bead) { b.Status = "in_progress" }), false, ""},
	{"deferred", with(workable, func(b *beads.Bead) { b.Status = "deferred" }), false, "unrefined"},
	{"closed", with(workable, func(b *beads.Bead) { b.Status = "closed" }), false, "already closed"},
	{"no acceptance criteria", with(workable, func(b *beads.Bead) { b.AcceptanceCriteria = "  \n" }), false, "no acceptance criteria"},
	{"blocked status", with(workable, func(b *beads.Bead) { b.Status = "blocked" }), false, "is blocked, not workable"},

	// /refine would move it out of the status that is the actual blocker.
	{"unworkable status outranks missing criteria", with(workable, func(b *beads.Bead) { b.Status, b.AcceptanceCriteria = "blocked", "" }), false, "is blocked, not workable"},
	{"blocked by a dependency", workable, false, "open dependency"},

	{"open epic, dependencies unasked", epic, false, ""},
	{"deferred epic", with(epic, func(b *beads.Bead) { b.Status = "deferred" }), false, ""},
	{"epic without acceptance criteria", with(epic, func(b *beads.Bead) { b.AcceptanceCriteria = "" }), false, ""},
	{"closed epic", with(epic, func(b *beads.Bead) { b.Status = "closed" }), false, "already closed"},
}

// Preparing is where a ticket that cannot be worked is refused, and the refusal
// says which rule it broke.
func TestPrepareVetsTheTicket(t *testing.T) {
	for _, tt := range vettings {
		t.Run(tt.name, func(t *testing.T) {
			ready := rows()
			if tt.ready {
				ready = rows(tt.bead)
			}
			r, _ := resolver(t, rows(tt.bead), ready)

			_, err := r.Prepare(worktree.Place{ID: tt.bead.ID, Name: tt.bead.ID})
			switch {
			case tt.want == "" && err != nil:
				t.Errorf("Prepare = %v; want the ticket workable", err)
			case tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)):
				t.Errorf("Prepare = %v; want it to mention %q", err, tt.want)
			}
		})
	}
}

// The economy of the query, stated as the property rather than as a list of the
// rules that hold it: a verdict both answers agree on cannot have turned on the
// question, so vetting must reach it without asking. Any rule added ahead of the
// dependency check answers to this without being named here.
func TestVettingAsksOnlyWhereTheAnswerDecides(t *testing.T) {
	for _, tt := range vettings {
		t.Run(tt.name, func(t *testing.T) {
			var asked int
			vet := func(ready bool) string {
				list := rows()
				if ready {
					list = rows(tt.bead)
				}
				r, ran := resolver(t, rows(tt.bead), list)
				_, err := r.Prepare(worktree.Place{ID: tt.bead.ID, Name: tt.bead.ID})
				asked += spawns(ran(), "bd ready")
				if err == nil {
					return ""
				}
				return err.Error()
			}
			if yes, no := vet(true), vet(false); yes == no && asked != 0 {
				t.Errorf("vetting asked whether the bead is ready %d times for a verdict the answer cannot change", asked)
			}
		})
	}
}

// A tracker that will not answer is not a ticket that cannot be worked: the
// failure is reported rather than turned into a refusal.
func TestVettingReportsAFailedQuery(t *testing.T) {
	shim(t, answers{list: rows(workable), ready: "not json at all"})
	r := New(t.TempDir(), t.TempDir(), pattern)

	_, err := r.Prepare(worktree.Place{ID: workable.ID, Name: workable.ID})
	if err == nil || strings.Contains(err.Error(), "open dependency") {
		t.Errorf("Prepare = %v; want the failed query reported rather than read as a blocked ticket", err)
	}
}

// The branch takes a slug of the title, which is what makes naming it the
// preparation's rather than the resolution's.
func TestPrepareNamesTheBranchFromTheTitle(t *testing.T) {
	bead := with(workable, func(b *beads.Bead) { b.Title = "Port work to Go" })
	r, _ := resolver(t, rows(bead), rows(bead))

	got, err := r.Prepare(worktree.Place{ID: bead.ID, Name: bead.ID})
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if got.Branch != "one-port-work-to-go" || got.Label != bead.Title {
		t.Errorf("Prepare = %+v; want the branch slugged from the title and the title on the place", got)
	}
}

func TestSlug(t *testing.T) {
	tests := []struct{ title, want string }{
		{"Port work to Go", "port-work-to-go"},
		{"Rate-limit /api/upload", "rate-limit-api-upload"},
		{"  Trailing punctuation!!  ", "trailing-punctuation"},
		{strings.Repeat("ab ", 30), strings.Repeat("ab-", 13) + "a"},
		{"—", ""},
	}
	for _, tt := range tests {
		if got := slug(tt.title); got != tt.want {
			t.Errorf("slug(%q) = %q, want %q", tt.title, got, tt.want)
		}
		if len(slug(tt.title)) > slugLen {
			t.Errorf("slug(%q) is longer than %d", tt.title, slugLen)
		}
	}
}

// An identifier is one bd lists, so a name no bead answers for is left to the
// resolvers after this one rather than carried to bd as an id to look up.
func TestIdentifyAnswersOnlyForABeadBdKnows(t *testing.T) {
	known := beads.Bead{ID: "bd-42", Title: "A ticket"}
	ran := shim(t, answers{list: rows(known)})
	r := New(t.TempDir(), t.TempDir(), pattern)

	got, err := r.Identify(known.ID, worktree.Open{})
	if err != nil {
		t.Fatalf("Identify(%q): %v", known.ID, err)
	}
	want := worktree.Place{ID: known.ID, Name: known.ID, Label: known.Title}
	if got != want {
		t.Errorf("Identify(%q) = %+v; want %+v", known.ID, got, want)
	}

	for _, id := range []string{"one", "1234", "anything-at-all"} {
		if _, err := r.Identify(id, worktree.Open{}); !errors.Is(err, worktree.ErrUnknown) {
			t.Errorf("Identify(%q) = %v; want it left to the next resolver", id, err)
		}
	}
	if n := spawns(ran(), "bd show"); n != 0 {
		t.Errorf("naming an identifier asked bd show %d times; want the listing to settle it", n)
	}
}

// A bd that will not answer is not a name it does not know: the refusal stops
// the run rather than leaving a ticket id to become a plain branch.
func TestIdentifyReportsATrackerThatWillNotList(t *testing.T) {
	shim(t, answers{})
	r := New(t.TempDir(), t.TempDir(), pattern)

	_, err := r.Identify("one", worktree.Open{})
	if err == nil || errors.Is(err, worktree.ErrUnknown) {
		t.Errorf(`Identify("one") = %v; want the failed listing reported rather than read as a bead bd does not know`, err)
	}
}

// A worktree is named by the longest id bd knows that owns its branch, so a
// branch of one-two's does not fall to one.
func TestIdentifyNamesAWorktreeByTheLongestId(t *testing.T) {
	known := []beads.Bead{
		{ID: "one", Title: "The first bead"},
		{ID: "one-two", Title: "The second bead"},
		{ID: "1234", Title: "The numbered bead"},
	}
	tests := []struct {
		name   string
		branch string
		want   string // the id, or "" for a worktree left to another resolver
		label  string
	}{
		{"a bare id", "one", "one", "The first bead"},
		{"an id and a title slug", "one-and-a-slug", "one", "The first bead"},
		{"the longest id wins", "one-two-three", "one-two", "The second bead"},
		// A numeric branch is still a ticket's: reading a pull request into it belongs
		// to the resolver that owns pull requests.
		{"a numeric id", "1234", "1234", "The numbered bead"},
		{"a branch no id owns", "some-branch", "", ""},
		{"detached", "", "", ""},
	}
	r, _ := resolver(t, rows(known...), rows())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.Identify("", worktree.Open{Path: "/wt", Branch: tt.branch})
			if tt.want == "" {
				if !errors.Is(err, worktree.ErrUnknown) {
					t.Fatalf("Identify(branch %q) = %+v, %v; want the worktree left to another resolver", tt.branch, got, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Identify(branch %q): %v", tt.branch, err)
			}
			want := worktree.Place{ID: tt.want, Name: tt.want, Branch: tt.branch, Label: tt.label}
			if got != want {
				t.Errorf("Identify(branch %q) = %+v; want %+v", tt.branch, got, want)
			}
		})
	}
}

// Asked whether an open worktree is a given ticket's, the branch pattern settles
// it first, and a longer id bd knows takes a branch that spells both.
func TestIdentifyConfirmsATicketAgainstAWorktree(t *testing.T) {
	known := rows(beads.Bead{ID: "bd-4"}, beads.Bead{ID: "bd-42"})
	tests := []struct {
		id, branch string
		want       bool
	}{
		{"bd-42", "bd-42", true},
		{"bd-42", "bd-42-port-work-to-go", true},
		{"bd-42", "bd-42.1-a-child", false},
		{"bd-4", "bd-42", false},
		{"bd-4", "bd-42-port-work-to-go", false},
		{"bd-42", "", false},
		{"bd-9", "bd-9-a-slug", true},
		// The branch retyped, which is what a listing prints: the pattern's empty-slug
		// arm takes it, so it reaches its own worktree without bd naming an id.
		{"bd-42-port-work-to-go", "bd-42-port-work-to-go", true},
	}
	r, _ := resolver(t, known, rows())
	for _, tt := range tests {
		_, err := r.Identify(tt.id, worktree.Open{Path: "/wt", Branch: tt.branch})
		if got := err == nil; got != tt.want {
			t.Errorf("Identify(%q, branch %q) = %v; want answered %v", tt.id, tt.branch, err, tt.want)
		}
		// Either way this is a worktree another resolver may claim, never a run to stop.
		if err != nil && !errors.Is(err, worktree.ErrUnknown) {
			t.Errorf("Identify(%q, branch %q) = %v; want unrecognition rather than a failure", tt.id, tt.branch, err)
		}
	}
}

// A bd that will not list names no worktree, so its rows lose their titles and
// the worktrees still list, adopted by whichever resolver answers for what is
// left. The identifier is what recognises one again, which is the reading that
// must not need the tracker at all.
func TestIdentifyWithoutTheTracker(t *testing.T) {
	shim(t, answers{})
	r := New(t.TempDir(), t.TempDir(), pattern)
	o := worktree.Open{Path: "/wt", Branch: "one-a-slug"}

	if _, err := r.Identify("", o); !errors.Is(err, worktree.ErrUnknown) {
		t.Errorf("Identify(branch alone) = %v; want the worktree left to another resolver", err)
	}
	got, err := r.Identify("one", o)
	if err != nil {
		t.Fatalf("Identify(one, its own branch): %v", err)
	}
	if got.ID != "one" || got.Branch != "one-a-slug" {
		t.Errorf("Identify = %+v; want the worktree recognised by its identifier", got)
	}
}

// A branch the pattern cannot name for the identifier is ruled out ahead of the
// listing, which is the one path a worktree already open must not pay bd for.
func TestIdentifyRulesOutABranchWithoutAskingBd(t *testing.T) {
	ran := shim(t, answers{list: rows(beads.Bead{ID: "one"})})
	r := New(t.TempDir(), t.TempDir(), pattern)

	o := worktree.Open{Path: "/wt", Branch: "some-other-branch"}
	if _, err := r.Identify("one", o); !errors.Is(err, worktree.ErrUnknown) {
		t.Fatalf("Identify = %v; want the worktree left to another resolver", err)
	}
	if n := spawns(ran(), "bd"); n != 0 {
		t.Errorf("ruling a branch out asked bd %d times; want the pattern to settle it alone", n)
	}
}

// A detached worktree has no branch for any id to own, so it is left to another
// resolver without the listing that could not have named it either way.
func TestIdentifyLeavesADetachedWorktreeWithoutAskingBd(t *testing.T) {
	ran := shim(t, answers{list: rows(beads.Bead{ID: "one"})})
	r := New(t.TempDir(), t.TempDir(), pattern)

	o := worktree.Open{Path: "/wt"}
	for _, id := range []string{"", "one"} {
		if _, err := r.Identify(id, o); !errors.Is(err, worktree.ErrUnknown) {
			t.Errorf("Identify(%q, detached) = %v; want the worktree left to another resolver", id, err)
		}
	}
	if n := spawns(ran(), "bd"); n != 0 {
		t.Errorf("identifying a detached worktree asked bd %d times; want the missing branch to settle it alone", n)
	}
}

// The picker's rows are what bd calls workable, kept whole, so entering one of
// them is vetted against the very snapshot that called it ready rather than by
// asking bd again.
func TestOfferHoldsTheRecordsItOffered(t *testing.T) {
	ran := shim(t, answers{ready: rows(workable), list: rows()})
	r := New(t.TempDir(), t.TempDir(), pattern)

	offered, err := r.Offer()
	if err != nil {
		t.Fatalf("Offer: %v", err)
	}
	want := worktree.Place{ID: workable.ID, Name: workable.ID, Label: workable.Title}
	if len(offered) != 1 || offered[0] != want {
		t.Fatalf("Offer = %+v; want %+v", offered, want)
	}

	if _, err := r.Prepare(offered[0]); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	out := ran()
	if shows, ready := spawns(out, "bd show"), spawns(out, "bd ready"); shows != 0 || ready != 1 {
		t.Errorf("picking a fresh ticket asked bd show %d times and bd ready %d; want the picker's own listing to serve both", shows, ready)
	}
}

// The full listing serves the vetting too: a bead it named is worked from that
// record, so preparing a fresh ticket asks bd for nothing of its own.
func TestPrepareVetsFromTheListing(t *testing.T) {
	// in_progress, so the vetting needs no readiness query either.
	listed := with(workable, func(b *beads.Bead) { b.Status = "in_progress" })
	ran := shim(t, answers{list: rows(listed)})
	r := New(t.TempDir(), t.TempDir(), pattern)

	if _, err := r.Prepare(worktree.Place{ID: listed.ID, Name: listed.ID}); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if n := spawns(ran(), "bd show"); n != 0 {
		t.Errorf("preparing a listed bead asked bd show %d times; want it vetted from the listing", n)
	}
}

// One resolver asks bd for each of its listings at most once, however many of
// its questions want them.
func TestOneRunAsksEachListingOnce(t *testing.T) {
	ran := shim(t, answers{list: rows(workable), ready: rows(workable)})
	r := New(t.TempDir(), t.TempDir(), pattern)

	for _, branch := range []string{"one", "one-a-slug", "one-and-another"} {
		if _, err := r.Identify("", worktree.Open{Path: "/wt", Branch: branch}); err != nil {
			t.Fatalf("Identify(branch %q): %v", branch, err)
		}
	}
	if _, err := r.Offer(); err != nil {
		t.Fatalf("Offer: %v", err)
	}
	if _, err := r.Prepare(worktree.Place{ID: workable.ID, Name: workable.ID}); err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	out := ran()
	if lists, ready := spawns(out, "bd list"), spawns(out, "bd ready"); lists > 1 || ready > 1 {
		t.Errorf("one run asked bd list %d times and bd ready %d; want each listing made at most once", lists, ready)
	}
}

// A configured pattern names the branch a new worktree checks out and finds that
// worktree again: the id is matched where the pattern puts it, so a prefix costs
// nothing and a ticket retitled since is still the branch's owner.
func TestConfiguredBranchPattern(t *testing.T) {
	bead := with(workable, func(b *beads.Bead) { b.ID, b.Title = "one-abc", "A new title" })
	repo := t.TempDir()
	testenv.Settings(t, "[branch]\nticket = \"feature/{{.ID}}-{{.Slug}}\"\n")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	shim(t, answers{list: rows(bead), ready: rows(bead)})
	r := New(repo, repo, cfg.Branch)

	got, err := r.Prepare(worktree.Place{ID: bead.ID, Name: bead.ID})
	if err != nil || got.Branch != "feature/one-abc-a-new-title" {
		t.Fatalf("Prepare = %+v, %v; want the configured pattern's branch", got, err)
	}
	// Made when the bead was titled otherwise, and recognised all the same.
	o := worktree.Open{Path: "/wt", Branch: "feature/one-abc-an-older-title"}
	named, err := r.Identify("", o)
	if err != nil || named.ID != bead.ID {
		t.Errorf("Identify(an older title's branch) = %+v, %v; want the ticket that owns it", named, err)
	}
}

// bd takes no fork point of its own, so the directory it is run in is what
// carries one: creating from inside another worktree forks the branch from the
// work under foot rather than from what the main checkout happens to have.
func TestCreateForksFromTheCurrentCheckout(t *testing.T) {
	shim(t, answers{})
	repo := testenv.InitRepo(t)
	main := testenv.Git(t, repo, "rev-parse", "HEAD")
	under := filepath.Join(repo, "trees", "under-foot")
	testenv.Git(t, repo, "worktree", "add", "-b", "under-foot", under)
	testenv.Git(t, under, "commit", "--allow-empty", "-m", "not on main yet")
	ahead := testenv.Git(t, under, "rev-parse", "HEAD")
	r := New(repo, under, pattern)

	path := filepath.Join(repo, "trees", "one")
	if err := r.Create(worktree.Place{ID: "one", Name: "one", Branch: "one"}, path); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if head := testenv.Git(t, repo, "rev-parse", "one"); head != ahead {
		t.Errorf("branch one is at %q; want the checkout under foot at %q rather than the main checkout's %q", head, ahead, main)
	}
}

// What this resolver tells a command about the worktree it resolved is the
// ticket's id and title, and nothing that would name a tracker.
func TestSupplyIsTheTicket(t *testing.T) {
	r := New(t.TempDir(), t.TempDir(), pattern)
	tree := worktree.Tree{Place: worktree.Place{ID: "one", Name: "one", Label: "A ticket"}, Path: "/wt"}

	vals, err := r.Supply(tree)
	if err != nil {
		t.Fatalf("Supply: %v", err)
	}
	if want := (worktree.Values{"ID": "one", "Title": "A ticket"}); !maps.Equal(vals, want) {
		t.Errorf("Supply = %v; want %v", vals, want)
	}
}

// A tracker work cannot reach is refused as the command bd was put and what it
// answered with: reaching for the tracker and failing is not the same as a
// repository with nothing ready, and the command is what the reader runs to find
// out which it was.
func TestARefusedListingNamesTheCommandItRan(t *testing.T) {
	testenv.Stubs(t, testenv.Stub{Name: "bd", Grumbles: "the database is not there", Exits: 1})
	r := New(t.TempDir(), t.TempDir(), pattern)

	places, err := r.Offer()
	if err == nil {
		t.Fatalf("Offer = %v; want the refusal handed on", places)
	}
	if head, tail := "bd ready ", ": the database is not there"; !strings.HasPrefix(err.Error(), head) || !strings.HasSuffix(err.Error(), tail) {
		t.Errorf("Offer = %v; want a refusal from %q to %q", err, head, tail)
	}
}

func with(b beads.Bead, f func(*beads.Bead)) beads.Bead {
	f(&b)
	return b
}

// rows renders what bd answers a listing query with.
func rows(list ...beads.Bead) string {
	out, err := json.Marshal(list)
	if err != nil {
		panic(err)
	}
	return string(out)
}

// answers is what the bd on PATH replies with, per subcommand.
type answers struct {
	list, ready string
}

// resolver is one answering over a bd that replies with these listings, and what
// that bd was asked to run.
func resolver(t *testing.T, all, ready string) (*Resolver, func() []string) {
	t.Helper()
	ran := shim(t, answers{list: all, ready: ready})
	return New(t.TempDir(), t.TempDir(), pattern), ran
}

// shim puts a bd on PATH ahead of the real one and hands back what it has been
// asked to run. bd is answered rather than run: the beads are the test's, and
// there is no tracker database here to hold them. A subcommand with no answer of
// its own replies nothing, which is a bd that will not list. Creating a worktree
// is the exception, carried out the way the real bd carries it out — git worktree
// add, run wherever bd stands — because that is what settles the fork point.
func shim(t *testing.T, a answers) func() []string {
	t.Helper()
	dir := t.TempDir()
	for sub, body := range map[string]string{"list": a.list, "ready": a.ready} {
		if body != "" {
			testenv.Write(t, filepath.Join(dir, "answer-"+sub), body)
		}
	}
	return testenv.Stubs(t, testenv.Stub{Name: "bd", Shell: fmt.Sprintf(`if [ "$1" = worktree ]; then
	exec git worktree add "$3" -b "$5"
fi
cat %q/answer-"$1" 2>/dev/null
`, dir)})
}

// spawns counts the invocations of one command in what the shim recorded.
func spawns(ran []string, prefix string) int {
	n := 0
	for _, cmd := range ran {
		if strings.HasPrefix(cmd, prefix) {
			n++
		}
	}
	return n
}
