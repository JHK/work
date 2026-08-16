// Package shim is the shell function work is called through: the file that
// function names for the worktree, and the worktree written back into it.
package shim

import (
	_ "embed"
	"fmt"
	"io"
	"os"
)

// CDFile is the environment variable the shim names its file in.
const CDFile = "WORK_CD_FILE"

// Fish is the function [work init fish] prints above the completions.
//
//go:embed work.fish
var Fish string

// Bash is the function [work init bash] and [work init zsh] print above the
// completions. It is written in the words both shells read.
//
//go:embed work.bash
var Bash string

// Answer hands the worktree back: into the file the shim named, else on out.
func Answer(dir string, out io.Writer) error {
	if file := os.Getenv(CDFile); file != "" {
		return os.WriteFile(file, []byte(dir+"\n"), 0o600)
	}
	_, err := fmt.Fprintln(out, dir)
	return err
}

// Forget takes the file out of the environment. It names one invocation, so
// whatever that invocation hands the terminal to must not inherit it.
func Forget() error { return os.Unsetenv(CDFile) }
