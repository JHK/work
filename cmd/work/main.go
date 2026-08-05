// Command work turns a ticket, a pull request, or an open worktree into a git
// worktree with a coding-agent session inside it.
package main

import (
	"os"

	"github.com/JHK/work-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
