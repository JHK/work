// Package shell answers with the worktree itself: the directory the shell that
// called work stands in from here.
package shell

import (
	"os"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/worktree"
)

const name = config.ShellOpener

// Opener hands the worktree back rather than running anything inside it.
type Opener struct{}

func (Opener) Name() string { return name }

// Open answers with the worktree, which is a handoff naming no command.
func (Opener) Open(t worktree.Tree) (worktree.Handoff, error) {
	// No chdir stands behind this handoff the way one stands behind a command, so a
	// worktree git still lists but nobody can enter is refused here or nowhere.
	if _, err := os.Stat(t.Path); err != nil {
		return worktree.Handoff{}, err
	}
	return worktree.Handoff{Dir: t.Path}, nil
}
