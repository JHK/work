// Command work turns a ticket, a pull request, or an open worktree into a git
// worktree with a coding-agent session inside it.
package main

import (
	"errors"
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "work:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	return errors.New("not implemented")
}
