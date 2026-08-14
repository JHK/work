package shell

import (
	"path/filepath"
	"testing"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/worktree"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// The action goes by the name the flag reaches it by and an [action] key spells
// it as.
func TestTheActionGoesByItsKeysName(t *testing.T) {
	if got := (Action{}).Name(); got != string(config.ActionShell) {
		t.Errorf("the action goes by %q; want %q", got, config.ActionShell)
	}
}

// The worktree itself is the answer: nothing is run in it, and nothing renders,
// so no value has to be supplied for one.
func TestOpenAnswersWithTheWorktree(t *testing.T) {
	dir := t.TempDir()
	tree := worktree.Tree{Place: worktree.Place{Name: "bd-42"}, Path: dir}

	got, err := (Action{}).Open(tree, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got.Dir != dir {
		t.Errorf("the answer is %q; want the worktree", got.Dir)
	}
	if !got.Directory() {
		t.Errorf("the answer runs %q; want nothing to run", got.Run)
	}
}

// A worktree git still lists but nobody can enter is refused. Nothing execs on
// this path, so an answer nobody checked would be a shell left where it was.
func TestOpenRefusesAWorktreeThatIsNotThere(t *testing.T) {
	tree := worktree.Tree{Place: worktree.Place{Name: "gone"}, Path: filepath.Join(t.TempDir(), "gone")}

	if _, err := (Action{}).Open(tree, nil); err == nil {
		t.Error("Open on a worktree that was pruned from under it: want the refusal")
	}
}
