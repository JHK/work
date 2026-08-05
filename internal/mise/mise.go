// Package mise grants a fresh worktree the config trust its sessions need.
package mise

import "os/exec"

// Trust marks the worktree's mise configs as trusted, letting mise find them
// itself rather than reproducing its discovery order here. Best effort: a
// missing mise, or a grant that fails, only means the session prompts.
func Trust(worktree string) {
	cmd := exec.Command("mise", "trust")
	cmd.Dir = worktree
	_ = cmd.Run()
}
