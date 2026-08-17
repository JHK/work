// Package shim is the shell function work is called through: the file that
// function names for the worktree, and the worktree written back into it.
package shim

import (
	_ "embed"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
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

// advice is the line a terminal reading the path is given.
const advice = "work: the shell integration is not sourced, so your shell stays where it is; see work init --help"

// Answer hands the worktree back: into the file the shim named, else on out,
// with one line on note naming work init where a terminal is reading that path.
func Answer(dir string, out, note io.Writer) error {
	if file := os.Getenv(CDFile); file != "" {
		return os.WriteFile(file, []byte(dir+"\n"), 0o600)
	}
	if _, err := fmt.Fprintln(out, dir); err != nil {
		return err
	}
	if terminal(out) {
		_, _ = fmt.Fprintln(note, advice)
	}
	return nil
}

// terminal reports whether w is the terminal itself rather than a file, a pipe
// or a device reading the path.
func terminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// Forget takes the file out of the environment. It names one invocation, so
// whatever that invocation hands the terminal to must not inherit it.
func Forget() error { return os.Unsetenv(CDFile) }
