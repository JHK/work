package config

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// What a user spells out, which no other test may derive from the code: the file
// name, and the branches the defaults name.
const (
	defaultDir  = ".worktrees"
	userRelPath = "work/config.toml"

	ticketBranch      = "bd-42-port-work-to-go"
	untitledBranch    = "bd-42"
	pullRequestBranch = "pr-7"
)

// The one file is layered over the defaults, and a key it does not set is the
// default's.
func TestLoadLayers(t *testing.T) {
	tests := []struct{ name, user, want string }{
		{"no file at all", "", defaultDir},
		{"the file", "mine", "mine"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := testenv.Home(t)
			if tt.user != "" {
				testenv.Write(t, filepath.Join(home, userRelPath), directory(tt.user))
			}

			got, err := Load()
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
	testenv.Settings(t, "[worktree]\n[branch]\n")

	got, err := Load()
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
	testenv.Settings(t, branch(`feature/{{.ID}}-{{.Slug}}`, `review/{{.Number}}`))

	c, err := Load()
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

// A zero Claude is the compiled-in commands, so an Env built without a loaded
// Config still names something to run.
func TestDefaultClaudeCommands(t *testing.T) {
	var c Claude
	l := worktree.Values{"Name": "bd-42", "Dir": "/w", "ID": "bd-42", "Title": "Port work to Go", "Number": "7", "Session": "s1"}

	tests := []struct {
		name string
		got  func() Command
		want []string
	}{
		{"start-ticket", c.StartTicket,
			[]string{"claude", "--permission-mode", "auto", "--name=bd-42: Port work to Go", "/start bd-42"},
		},
		{"start-pull-request", c.StartPullRequest,
			[]string{"claude", "--name=PR #7"},
		},
		{"start-session", c.StartSession,
			[]string{"claude", "--permission-mode", "auto", "--name=bd-42"},
		},
		{"resume-session", c.ResumeSession,
			[]string{"claude", "--resume", "s1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.got().Render(l)
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// An unset [action] key names an action even on a Config that never reached
// Load, and names one no system has to be on for: what these keys fall to is
// what a machine that configured nothing opens on, and the actions it never
// switched on are what loading refuses.
func TestDefaultActions(t *testing.T) {
	var a Action
	if got := a.Create(); got != ActionShell {
		t.Errorf("Create() = %q, want %q", got, ActionShell)
	}
	if got := a.Enter(); got != ActionShell {
		t.Errorf("Enter() = %q, want %q", got, ActionShell)
	}
	d := Default()
	for _, name := range []ActionName{d.Action.Create(), d.Action.Enter()} {
		if err := name.validate(d.actionNames()); err != nil {
			t.Errorf("an unset [action] key opens on %q, which a settings file has to switch on: %v", name, err)
		}
	}
}

// Each key is read on its own, so a file may name one action and leave the
// other to the default.
func TestConfiguredActions(t *testing.T) {
	testenv.Settings(t, "[claude]\nenabled = true\n[action]\nenter = \"claude\"\n")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := c.Action.Enter(); got != ActionClaude {
		t.Errorf("action.enter = %q, want %q", got, ActionClaude)
	}
	if got := c.Action.Create(); got != ActionShell {
		t.Errorf("action.create = %q, want the default %q", got, ActionShell)
	}
}

// A command is the user's, whatever it launches, and nothing requires it to be
// an agent: one naming no session value at all is a plain command line.
func TestConfiguredClaudeCommands(t *testing.T) {
	testenv.Settings(t, `[claude]
start-ticket = ["agent", "--session={{.Name}} in {{.Dir}}", "work {{.ID}}"]
start-pull-request = ["make", "review"]
resume-session = ["agent", "--continue"]
`)

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	l := worktree.Values{"Name": "bd-42", "Dir": "/w", "ID": "bd-42", "Title": "Port work to Go", "Number": "7", "Session": ""}

	got, err := c.Claude.StartTicket().Render(l)
	if err != nil {
		t.Fatalf("StartTicket: %v", err)
	}
	if want := []string{"agent", "--session=bd-42 in /w", "work bd-42"}; !reflect.DeepEqual(got, want) {
		t.Errorf("StartTicket() = %q, want %q", got, want)
	}

	got, err = c.Claude.StartPullRequest().Render(l)
	if err != nil {
		t.Fatalf("StartPullRequest: %v", err)
	}
	if want := []string{"make", "review"}; !reflect.DeepEqual(got, want) {
		t.Errorf("StartPullRequest() = %q, want %q", got, want)
	}
}

// A first element rendering to nothing leaves no command, which only a render
// can find out: the value it needed is one the invocation carries.
func TestClaudeCommandRendersNothing(t *testing.T) {
	testenv.Settings(t, "[claude]\nresume-session = [\"{{.Session}}\", \"--continue\"]\n")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, err := c.Claude.ResumeSession().Render(worktree.Values{"Name": "", "Dir": "", "Session": "s1"}); err != nil {
		t.Fatalf("ResumeSession with a session: %v", err)
	} else if got[0] != "s1" {
		t.Errorf("ResumeSession() = %q, want the session as the command", got)
	}

	_, err = c.Claude.ResumeSession().Render(worktree.Values{"Name": "", "Dir": "", "Session": ""})
	if err == nil || !strings.Contains(err.Error(), resumeSessionKey) {
		t.Errorf("ResumeSession() with no session = %v, want it to name %q", err, resumeSessionKey)
	}
}

// A command replaces the default whole rather than element by element, so one
// shorter than the default leaves no tail of it behind.
func TestLoadReplacesACommandWhole(t *testing.T) {
	testenv.Settings(t, "[claude]\nstart-ticket = [\"ours\", \"{{.ID}}\"]\n")

	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, err := c.Claude.StartTicket().Render(worktree.Values{"Name": "", "Dir": "", "ID": "bd-42", "Title": ""})
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

// No system runs until a file asks for it, and a system's own table is what asks
// for that one system.
func TestSystemsRunOnlyWhereSwitchedOn(t *testing.T) {
	systems := []struct {
		name string
		on   func(Config) bool
	}{
		{"github", func(c Config) bool { return c.Github.Enabled }},
		{"beads", func(c Config) bool { return c.Beads.Enabled }},
		{"mise", func(c Config) bool { return c.Mise.Enabled }},
		{"claude", func(c Config) bool { return c.Claude.Enabled }},
	}
	for _, s := range systems {
		if s.on(Default()) {
			t.Errorf("%s runs with nothing configured; want the worktrees alone until a file asks for it", s.name)
		}
	}
	for _, tt := range systems {
		t.Run(tt.name, func(t *testing.T) {
			testenv.Settings(t, "["+tt.name+"]\nenabled = true\n")

			c, err := Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			for _, s := range systems {
				if got, want := s.on(c), s.name == tt.name; got != want {
					t.Errorf("switching %s on left %s %v; want %v", tt.name, s.name, got, want)
				}
			}
		})
	}
}

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
		{"a system work does not have", "[linear]\nenabled = true\n", "unknown setting"},
		{"an unknown action key", "[action]\nopen = \"shell\"\n", "unknown setting"},
		{"an action nothing goes by", "[action]\ncreate = \"launcher\"\n", "is not an action"},
		{"an action named for its flag", "[action]\nenter = \"--shell\"\n", "is not an action"},
		{"the unnamed action, which is no action", "[action]\nenter = \"unnamed\"\n", "is not an action"},
		{"an action that is not a string", "[action]\ncreate = 3\n", "create"},
		{"an action in another case", "[action]\nenter = \"Shell\"\n", "is not an action"},
		// A file written before a rename is told the new spelling rather than that
		// what it names is unknown.
		{"an action under the name it used to go by", "[action]\ncreate = \"agent\"\n", `"agent" is now "claude"`},
		{"a table under the name it used to go by", "[agent]\nstart-ticket = [\"claude\"]\n", "the [agent] table is now [claude]"},
		{"an unknown command key", "[claude]\nstart = [\"claude\"]\n", "unknown setting"},
		{"a command that is not a list", "[claude]\nstart-ticket = \"claude\"\n", "list of command line arguments"},
		{"a list of something other than strings", "[claude]\nstart-ticket = [1, 2]\n", "list of command line arguments"},
		{"a template that does not parse", "[claude]\nstart-ticket = [\"claude\", \"{{.ID\"]\n", startTicketKey},
		{"no command at all", "[claude]\nstart-ticket = []\n", "names no command"},
		{"a value the key does not have", "[claude]\nstart-ticket = [\"claude\", \"{{.Number}}\"]\n", "{{.Title}}"},
		{"a value no key has", "[claude]\nresume-session = [\"claude\", \"{{.Branch}}\"]\n", resumeSessionKey},
		// The two work once placed itself, and now has no more than any other name.
		{"a model or an effort", "[claude]\nstart-ticket = [\"claude\", \"--model={{.Model}}\", \"--effort={{.Effort}}\"]\n", startTicketKey},
		// Only the arm a target with a session reaches names it.
		{"a value named inside a branch", "[claude]\nresume-session = [\"claude\", \"{{with .Session}}{{$.Branch}}{{end}}\"]\n", resumeSessionKey},
		// The whole table went with the commands that were under it: the shell action
		// hands back the worktree now, and the editor and the diff are gone.
		{"the table of commands to open on", "[open]\nshell = [\"fish\"]\neditor = [\"vi\", \"{{.Dir}}\"]\n", "unknown setting open"},
		{"the editor action", "[action]\ncreate = \"editor\"\n", "action.create"},
		{"the diff action", "[action]\nenter = \"diff\"\n", "action.enter"},
		{"the screen, which was never a command", "[action]\ncreate = \"ask\"\n", "action.create"},
		// An action work has, of a system this machine did not switch on.
		{"an action nothing here is on for", "[action]\ncreate = \"claude\"\n", "is not an action"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testenv.Settings(t, tt.body)

			_, err := Load()
			if err == nil {
				t.Fatalf("Load(%q) = no error, want one", tt.body)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Load(%q) = %v, want it to name %q", tt.body, err, tt.want)
			}
		})
	}
}

// A refusal names the file it was read from, which is where the reader goes to
// put it right.
func TestLoadNamesTheFileAtFault(t *testing.T) {
	user := testenv.Settings(t, directory("/tmp/trees"))

	_, err := Load()
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
