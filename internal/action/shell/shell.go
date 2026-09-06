// Package shell answers with the worktree itself: the directory the shell that
// called work stands in from here.
package shell

import (
	"os"

	"github.com/JHK/work-cli/internal/worktree"
)

// Name is what this action goes by.
const Name = "shell"

// Handback hands the worktree back rather than running anything inside it.
type Handback struct{}

func (Handback) Name() worktree.IntegrationName { return Name }

// OnCreated has nothing to do: a worktree is handed back however it came about.
func (Handback) OnCreated(worktree.Tree) error { return nil }

// Open answers with the worktree, which is a handoff naming no command.
func (Handback) Open(t worktree.Tree) (worktree.Handoff, error) {
	// No chdir stands behind this handoff the way one stands behind a command, so a
	// worktree git still lists but nobody can enter is refused here or nowhere.
	if _, err := os.Stat(string(t.Path)); err != nil {
		return worktree.Handoff{}, err
	}
	return worktree.Handoff{Dir: t.Path}, nil
}
