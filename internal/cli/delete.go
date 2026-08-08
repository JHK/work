package cli

import (
	"fmt"

	"github.com/JHK/work-cli/internal/work"
)

// remove takes a worktree away and says what went. It hands over to nothing:
// the invocation does its work and exits.
func remove(env work.Env, o options, target string) error {
	c, err := candidate(env, o, target)
	if err != nil {
		return err
	}
	d, err := env.Delete(c, o.force)
	if err != nil {
		return err
	}
	fmt.Println("removed worktree", d.Path)
	if d.Branch != "" {
		fmt.Println("deleted branch", d.Branch)
	}
	return nil
}
