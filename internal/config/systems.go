package config

// These are the tables a settings file switches a system on in, and also the
// names the implementations answer with. Nothing in the compiler holds the two
// spellings together.
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

// SystemKey is what switches a system on: its table, and the one key in it.
func SystemKey(name string) string { return name + ".enabled" }
