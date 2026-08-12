package config

// The systems are what work reaches for beyond git: a tracker, a forge, a tool,
// an agent. None runs until its own table asks for it, so a repository says which
// of them it works with: one filing its work in beads turns that on and is
// offered its tickets, and one handing its worktrees to the agent turns claude on.
//
// These are the tables, which are also the names the implementations answer
// with: cmd/work wires each system under the table that names it, and nothing in
// the compiler holds the two spellings together. The agent's is the action's own
// name, one system filling one seam under one name.
const (
	githubSystem = "github"
	beadsSystem  = "beads"
	miseSystem   = "mise"
	claudeSystem = string(ActionClaude)
)

// Github is the forge's table: the repository's pull requests.
type Github struct {
	Enabled bool
}

// Beads is the tracker's table, and answers for both halves of it: resolving a
// ticket, and claiming the one a worktree was made for.
type Beads struct {
	Enabled bool
}

// Mise is the tool's table: the trust a fresh worktree's configs are granted.
type Mise struct {
	Enabled bool
}

// SystemNames are the systems a settings file can switch on, under the tables
// they go by.
func SystemNames() []string {
	return []string{githubSystem, beadsSystem, miseSystem, claudeSystem}
}

// SystemKey is the key that puts a system back: the system's own table, and the
// one key in it that every system has. It is how a refusal names what would have
// answered.
func SystemKey(name string) string { return name + ".enabled" }

// Shipped is [Default] with every system work has an implementation for on,
// which is not what any repository runs: that is Default and whatever it asked
// for. The command line spells its flags from this, so one flag set serves every
// repository and a flag whose system is off is refused when it is given rather
// than missing from --help.
func Shipped() Config {
	c := Default()
	c.Github.Enabled, c.Beads.Enabled, c.Mise.Enabled, c.Claude.Enabled = true, true, true, true
	return c
}
