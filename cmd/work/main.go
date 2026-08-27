// Command work turns a ticket, a pull request, or an open worktree into a git
// worktree, and hands it to what that worktree opens on.
package main

import (
	"log/slog"
	"os"

	"github.com/JHK/work-cli/internal/cli"
	"github.com/JHK/work-cli/internal/wiring"
)

// version is stamped by the mise build tasks; a plain go build keeps the default.
var version = "dev"

func main() {
	logLevel := cli.LogLevel()
	slog.SetDefault(cli.Logger(os.Stderr, logLevel))
	os.Exit(cli.Execute(version, wiring.Wire, os.Args[1:], os.Stdout, logLevel))
}
