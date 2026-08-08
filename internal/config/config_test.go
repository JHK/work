package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// What a user spells out, which no other test may derive from the code: the two
// file names, and the branches the defaults name.
const (
	repoFile    = ".work.toml"
	defaultDir  = ".worktrees"
	userRelPath = "work/config.toml"

	ticketBranch      = "bd-42-port-work-to-go"
	untitledBranch    = "bd-42"
	pullRequestBranch = "pr-7"
)

// Both files are layered over the defaults, the repository's on top.
func TestLoadLayers(t *testing.T) {
	tests := []struct {
		name       string
		user, repo string
		want       string
	}{
		{"neither", "", "", defaultDir},
		{"the user alone", "mine", "", "mine"},
		{"the repository alone", "", "ours", "ours"},
		{"the repository over the user", "mine", "ours", "ours"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, home := t.TempDir(), t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", home)
			if tt.user != "" {
				write(t, filepath.Join(home, userRelPath), directory(tt.user))
			}
			if tt.repo != "" {
				write(t, filepath.Join(repo, repoFile), directory(tt.repo))
			}

			got, err := Load(repo)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got.Worktree.Directory != tt.want {
				t.Errorf("worktree.directory = %q, want %q", got.Worktree.Directory, tt.want)
			}
		})
	}
}

// A file that sets one key leaves the rest to the layer below it, so a table
// present but empty changes nothing.
func TestLoadLeavesUnnamedKeys(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	write(t, filepath.Join(repo, repoFile), "[worktree]\n[branch]\n")

	got, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// A pattern is identified by what it matches as, which is derived from the
	// whole of it.
	if !reflect.DeepEqual(got.Worktree, Default().Worktree) ||
		got.Branch.ticket().matcher != defaults.ticket().matcher ||
		got.Branch.pullRequest().matcher != defaults.pullRequest().matcher {
		t.Errorf("Load() = %+v, want the defaults", got)
	}
}

// The compiled-in patterns name the branches work named before they were
// settings, and read those branches back. A zero Branch is those patterns, so an
// Env built without a loaded Config still names one.
func TestDefaultBranches(t *testing.T) {
	var b Branch
	if got := b.Ticket("bd-42", "port-work-to-go"); got != ticketBranch {
		t.Errorf("Ticket() = %q, want %q", got, ticketBranch)
	}
	// A title that slugs to nothing leaves the id alone.
	if got := b.Ticket("bd-42", ""); got != untitledBranch {
		t.Errorf("Ticket() untitled = %q, want %q", got, untitledBranch)
	}
	if got := b.PullRequest("7"); got != pullRequestBranch {
		t.Errorf("PullRequest() = %q, want %q", got, pullRequestBranch)
	}

	for _, branch := range []string{ticketBranch, untitledBranch} {
		if !b.Owns("bd-42", branch) {
			t.Errorf("Owns(bd-42, %q) = false, want true", branch)
		}
	}
	// A shorter id is not a prefix of a longer one's branch.
	if b.Owns("bd-4", ticketBranch) {
		t.Errorf("Owns(bd-4, %q) = true, want false", ticketBranch)
	}
	if got, ok := b.NumberIn(pullRequestBranch); !ok || got != "7" {
		t.Errorf("NumberIn(%q) = %q, %v; want 7", pullRequestBranch, got, ok)
	}
	if got, ok := b.NumberIn(ticketBranch); ok {
		t.Errorf("NumberIn(%q) = %q, want no pull request", ticketBranch, got)
	}
}

// A pattern is the matcher too: the identifier is filled in and every other
// value stands for anything, so a prefix costs nothing and a ticket retitled
// after its worktree was made still finds it.
func TestConfiguredBranchesMatch(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	write(t, filepath.Join(repo, repoFile), branch(`feature/{{.ID}}-{{.Slug}}`, `review/{{.Number}}`))

	c, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := c.Branch.Ticket("bd-42", "port-work-to-go"); got != "feature/bd-42-port-work-to-go" {
		t.Errorf("Ticket() = %q, want the pattern's branch", got)
	}
	for _, b := range []string{"feature/bd-42-port-work-to-go", "feature/bd-42-anything-at-all"} {
		if !c.Branch.Owns("bd-42", b) {
			t.Errorf("Owns(bd-42, %q) = false, want true", b)
		}
	}
	// The prefix is part of the branch, so what lacks it is nobody's.
	if c.Branch.Owns("bd-42", "bd-42-port-work-to-go") {
		t.Error("Owns() = true for a branch without the configured prefix")
	}

	if got := c.Branch.PullRequest("7"); got != "review/7" {
		t.Errorf("PullRequest() = %q, want the pattern's branch", got)
	}
	if got, ok := c.Branch.NumberIn("review/7"); !ok || got != "7" {
		t.Errorf("NumberIn(review/7) = %q, %v; want 7", got, ok)
	}
	if got, ok := c.Branch.NumberIn(pullRequestBranch); ok {
		t.Errorf("NumberIn(%q) = %q, want no pull request", pullRequestBranch, got)
	}
}

// A zero Agent is the compiled-in commands, so an Env built without a loaded
// Config still names something to run.
func TestDefaultAgentCommands(t *testing.T) {
	var a Agent
	l := Launch{Name: "bd-42", Dir: "/w", ID: "bd-42", Title: "Port work to Go", Number: "7", Session: "s1"}

	tests := []struct {
		name string
		got  func(Launch) ([]string, error)
		want []string
	}{
		{"start-ticket", a.StartTicket,
			[]string{"claude", "--permission-mode", "auto", "--name=bd-42: Port work to Go", "/start bd-42"},
		},
		{"start-pull-request", a.StartPullRequest,
			[]string{"claude", "--name=PR #7"},
		},
		{"start-session", a.StartSession,
			[]string{"claude", "--permission-mode", "auto", "--name=bd-42"},
		},
		{"resume-session", a.ResumeSession,
			[]string{"claude", "--resume", "s1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.got(l)
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// A zero Open is the compiled-in commands: what the environment named for the
// two that read it, and git for the diff.
func TestDefaultOpenCommands(t *testing.T) {
	var o Open
	l := Launch{Name: "bd-42", Dir: "/w", Shell: "/usr/bin/fish", Editor: "gvim", Base: "abc123"}

	got, err := o.Shell(l)
	if want := []string{"/usr/bin/fish"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Shell() = %q, %v; want %q", got, err, want)
	}
	got, err = o.Editor(l)
	if want := []string{"gvim", "/w"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Editor() = %q, %v; want %q", got, err, want)
	}
	got, err = o.Diff(l)
	if want := []string{"git", "diff", "abc123"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Diff() = %q, %v; want %q", got, err, want)
	}
}

// An unset [action] key is what work did before either was a setting, so a
// Config that never reached Load names an action either way.
func TestDefaultActions(t *testing.T) {
	var a Action
	if got := a.Create(); got != ActionAgent {
		t.Errorf("Create() = %q, want %q", got, ActionAgent)
	}
	if got := a.Enter(); got != ActionShell {
		t.Errorf("Enter() = %q, want %q", got, ActionShell)
	}
}

// Each key is read on its own, so a file may name one action and leave the
// other to the default.
func TestConfiguredActions(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	write(t, filepath.Join(repo, repoFile), "[action]\nenter = \"editor\"\n")

	c, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := c.Action.Enter(); got != ActionEditor {
		t.Errorf("action.enter = %q, want %q", got, ActionEditor)
	}
	if got := c.Action.Create(); got != ActionAgent {
		t.Errorf("action.create = %q, want the default %q", got, ActionAgent)
	}
}

// A configured diff replaces the default whole, and places the base wherever the
// tool it names wants it.
func TestConfiguredDiffCommand(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	write(t, filepath.Join(repo, repoFile), "[open]\ndiff = [\"difft\", \"--\", \"{{.Base}}\", \"{{.Dir}}\"]\n")

	c, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, err := c.Open.Diff(Launch{Dir: "/w", Base: "abc123"})
	if want := []string{"difft", "--", "abc123", "/w"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Diff() = %q, %v; want %q", got, err, want)
	}
}

// A command is the user's, whatever it launches, and nothing requires it to be
// an agent: one naming no session value at all is a plain command line.
func TestConfiguredAgentCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	write(t, filepath.Join(home, userRelPath), `[agent]
start-ticket = ["agent", "--session={{.Name}} in {{.Dir}}", "work {{.ID}}"]
start-pull-request = ["make", "review"]
resume-session = ["agent", "--continue"]
`)

	c, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	l := Launch{Name: "bd-42", Dir: "/w", ID: "bd-42", Title: "Port work to Go", Number: "7"}

	got, err := c.Agent.StartTicket(l)
	if err != nil {
		t.Fatalf("StartTicket: %v", err)
	}
	if want := []string{"agent", "--session=bd-42 in /w", "work bd-42"}; !reflect.DeepEqual(got, want) {
		t.Errorf("StartTicket() = %q, want %q", got, want)
	}

	got, err = c.Agent.StartPullRequest(l)
	if err != nil {
		t.Fatalf("StartPullRequest: %v", err)
	}
	if want := []string{"make", "review"}; !reflect.DeepEqual(got, want) {
		t.Errorf("StartPullRequest() = %q, want %q", got, want)
	}
}

// A first element rendering to nothing leaves no command, which only a render
// can find out: the value it needed is one the invocation carries.
func TestAgentCommandRendersNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	write(t, filepath.Join(home, userRelPath), "[agent]\nresume-session = [\"{{.Session}}\", \"--continue\"]\n")

	c, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, err := c.Agent.ResumeSession(Launch{Session: "s1"}); err != nil {
		t.Fatalf("ResumeSession with a session: %v", err)
	} else if got[0] != "s1" {
		t.Errorf("ResumeSession() = %q, want the session as the command", got)
	}

	_, err = c.Agent.ResumeSession(Launch{})
	if err == nil || !strings.Contains(err.Error(), resumeSessionKey) {
		t.Errorf("ResumeSession() with no session = %v, want it to name %q", err, resumeSessionKey)
	}
}

// Every table layers the same way, commands included: the repository's file sets
// one the user's also set, and wins.
func TestLoadLayersCommands(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	write(t, filepath.Join(home, userRelPath), "[agent]\nstart-ticket = [\"mine\", \"--flag\", \"{{.ID}}\"]\n")
	// Shorter than the user's, so a command layered element by element rather than
	// replaced whole would leave the user's tail behind.
	write(t, filepath.Join(repo, repoFile), "[agent]\nstart-ticket = [\"ours\", \"{{.ID}}\"]\n")

	c, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, err := c.Agent.StartTicket(Launch{ID: "bd-42"})
	if err != nil {
		t.Fatalf("StartTicket: %v", err)
	}
	if want := []string{"ours", "bd-42"}; !reflect.DeepEqual(got, want) {
		t.Errorf("StartTicket() = %q, want %q", got, want)
	}
}

// An unset Directory is the default, so an Env built without a loaded Config
// still puts a worktree where one belongs rather than in the repository root.
func TestZeroWorktreeIsTheDefault(t *testing.T) {
	if got := (Worktree{}).Dir(); got != defaultDir {
		t.Errorf("Worktree{}.Dir() = %q, want %q", got, defaultDir)
	}
}

// A value is refused for what it carries, not for the file it came from, so the
// repository's file is judged by the same rules as the user's.
func TestLoadRefusals(t *testing.T) {
	tests := []struct{ name, body, want string }{
		{"an unknown key", "[worktree]\ndirectry = \"trees\"\n", "unknown setting"},
		{"an unknown table", "[nonsense]\nkey = 1\n", "unknown setting"},
		{"a value of the wrong type", "[worktree]\ndirectory = 3\n", "directory"},
		{"a key spelled in another case", "[worktree]\nDirectory = \"trees\"\n", "unknown setting"},
		{"a table spelled in another case", "[Worktree]\ndirectory = \"trees\"\n", "unknown setting"},
		{"a directory outside the repository", directory("../trees"), "not a directory inside"},
		{"an absolute directory", directory("/tmp/trees"), "not a directory inside"},
		{"the repository root, unnamed", directory(""), "not a directory inside"},
		{"the repository root, as a dot", directory("."), "not a directory inside"},
		{"the repository root, with a trailing slash", directory("./"), "not a directory inside"},
		{"the repository root, by traversal", directory("trees/.."), "not a directory inside"},
		{"git's own directory", directory(".git"), "git's own directory"},
		{"a directory under git's own", directory(".git/worktrees"), "git's own directory"},
		{"a pattern that does not parse", "[branch]\nticket = \"{{.ID\"\n", "branch.ticket"},
		{"a pattern that is not a string", "[branch]\nticket = 3\n", "ticket"},
		{"a ticket pattern without its id", "[branch]\nticket = \"feature/{{.Slug}}\"\n", "places no {{.ID}}"},
		// The pattern places the id, but only where a ticket with no slug would not
		// reach, and that ticket's branch would then stand for every ticket.
		{"an id only some tickets reach", "[branch]\nticket = \"{{with .Slug}}{{$.ID}}-{{.}}{{end}}\"\n", "places no {{.ID}}"},
		{"a pull request pattern without its number", "[branch]\npull-request = \"pr-{{.ID}}\"\n", "{{.Number}}"},
		{"a branch opening with a dash", "[branch]\nticket = \"-{{.ID}}\"\n", "dash"},
		{"an unknown action key", "[action]\nopen = \"shell\"\n", "unknown setting"},
		{"an action nothing goes by", "[action]\ncreate = \"launcher\"\n", "is not an action"},
		{"an action named for its flag", "[action]\nenter = \"--shell\"\n", "is not an action"},
		{"the unnamed action, which is no action", "[action]\nenter = \"unnamed\"\n", "is not an action"},
		{"an action that is not a string", "[action]\ncreate = 3\n", "create"},
		{"an action in another case", "[action]\nenter = \"Shell\"\n", "is not an action"},
		{"an unknown command key", "[agent]\nstart = [\"claude\"]\n", "unknown setting"},
		{"a command that is not a list", "[agent]\nstart-ticket = \"claude\"\n", "list of command line arguments"},
		{"a list of something other than strings", "[agent]\nstart-ticket = [1, 2]\n", "list of command line arguments"},
		{"a template that does not parse", "[agent]\nstart-ticket = [\"claude\", \"{{.ID\"]\n", startTicketKey},
		{"no command at all", "[agent]\nstart-ticket = []\n", "names no command"},
		{"a value the key does not have", "[agent]\nstart-ticket = [\"claude\", \"{{.Number}}\"]\n", "{{.Title}}"},
		{"a value no key has", "[agent]\nresume-session = [\"claude\", \"{{.Branch}}\"]\n", resumeSessionKey},
		// The two work once placed itself, and now has no more than any other name.
		{"a model or an effort", "[agent]\nstart-ticket = [\"claude\", \"--model={{.Model}}\", \"--effort={{.Effort}}\"]\n", startTicketKey},
		// Only the arm a target with a session reaches names it.
		{"a value named inside a branch", "[agent]\nresume-session = [\"claude\", \"{{with .Session}}{{$.Branch}}{{end}}\"]\n", resumeSessionKey},
		// Each of the three carries its own value alone, so none can place another's.
		{"the editor named by the shell", "[open]\nshell = [\"{{.Editor}}\"]\n", shellKey},
		{"the shell named by the editor", "[open]\neditor = [\"{{.Shell}}\", \"{{.Dir}}\"]\n", editorKey},
		{"the base named by the shell", "[open]\nshell = [\"git\", \"diff\", \"{{.Base}}\"]\n", shellKey},
		{"the editor named by the diff", "[open]\ndiff = [\"{{.Editor}}\", \"{{.Base}}\"]\n", diffKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			write(t, filepath.Join(repo, repoFile), tt.body)

			_, err := Load(repo)
			if err == nil {
				t.Fatalf("Load(%q) = no error, want one", tt.body)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Load(%q) = %v, want it to name %q", tt.body, err, tt.want)
			}
		})
	}
}

// Each file is decoded on its own, so the refusals above are the user's too.
func TestLoadRefusesTheUsersFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	write(t, filepath.Join(home, userRelPath), "[agent]\nstart = [\"claude\"]\n")

	_, err := Load(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "unknown setting") {
		t.Errorf("Load() = %v, want the user's unknown key refused", err)
	}
}

// The containment check is not lexical: a repository is cloned with its file
// and its symlinks, and a checkout may not land outside the repository.
func TestLoadRefusesADirectorySymlinkedOut(t *testing.T) {
	repo, outside := t.TempDir(), t.TempDir()
	if err := os.Symlink(outside, filepath.Join(repo, "trees")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	write(t, filepath.Join(repo, repoFile), directory("trees"))

	_, err := Load(repo)
	if err == nil || !strings.Contains(err.Error(), "resolves outside") {
		t.Errorf("Load() = %v, want the escaping symlink refused", err)
	}
}

// A repository reached through a symlink is still its own containing directory.
func TestLoadAllowsASymlinkedRepository(t *testing.T) {
	real, link := t.TempDir(), filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(real, "trees"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	write(t, filepath.Join(real, repoFile), directory("trees"))

	if _, err := Load(link); err != nil {
		t.Errorf("Load() = %v, want a repository behind a symlink to load", err)
	}
}

// A value the repository replaces is never the one work uses, so it is not the
// one judged: only what the merge arrives at has to be usable.
func TestLoadValidatesTheMergedValue(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	write(t, filepath.Join(home, userRelPath), directory("/tmp/trees"))
	write(t, filepath.Join(repo, repoFile), directory("trees"))

	got, err := Load(repo)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Worktree.Directory != "trees" {
		t.Errorf("worktree.directory = %q, want %q", got.Worktree.Directory, "trees")
	}
}

// Two files can carry the same unusable value, so the message names the one that
// did rather than leaving the reader to guess.
func TestLoadNamesTheFileAtFault(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	user := filepath.Join(home, userRelPath)
	write(t, user, directory("/tmp/trees"))

	_, err := Load(repo)
	if err == nil || !strings.Contains(err.Error(), user) {
		t.Errorf("Load() = %v, want it to name %q", err, user)
	}
}

func directory(dir string) string {
	return fmt.Sprintf("[worktree]\ndirectory = %q\n", dir)
}

func branch(ticket, pullRequest string) string {
	return fmt.Sprintf("[branch]\nticket = %q\npull-request = %q\n", ticket, pullRequest)
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
