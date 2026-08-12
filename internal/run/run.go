// Package run spawns the command-line tools the adapters read their state from.
package run

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var (
	// warnings is where a tool the machine does not have is said to be missing.
	warnings io.Writer = os.Stderr

	// gone are the commands this run has already found missing.
	gone sync.Map
)

// Output runs bin in dir and returns its stdout, trimmed of the trailing
// newline. A tool that fails diagnoses itself better than its exit status does,
// so its first line of stderr becomes the message.
//
// A tool the machine does not have is out for the rest of the run: the first
// question to it fails and every later one fails with it, unasked. One listing
// and every worktree behind it then cost one look at PATH rather than one each.
func Output(dir, bin string, args ...string) (string, error) {
	if _, missing := gone.Load(bin); missing {
		return "", absent(asked(bin, args), bin)
	}

	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		what := asked(bin, args)
		if errors.Is(err, exec.ErrNotFound) {
			return "", absent(what, bin)
		}
		if msg, _, _ := strings.Cut(strings.TrimSpace(stderr.String()), "\n"); msg != "" {
			return "", fmt.Errorf("%s: %s", what, msg)
		}
		return "", fmt.Errorf("%s: %w", what, err)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// asked is the tool and the question put to it, which is how a refusal names
// what work was doing.
func asked(bin string, args []string) string {
	if len(args) == 0 {
		return bin
	}
	return bin + " " + args[0]
}

// absent is a tool work reached for and the machine does not have. It is said
// here, where the reaching happened, because that is the only place it is always
// known: a listing that will not answer costs its own rows and is never reported,
// so a system missing its tool would otherwise go quiet rather than say why. The
// caller is handed the same sentence, for the paths that do report.
func absent(what, bin string) error {
	if _, before := gone.LoadOrStore(bin, true); !before {
		// Beside whatever the invocation prints, so a stderr that will not take it costs
		// nothing that was asked for.
		_, _ = fmt.Fprintf(warnings, "work: %s is not on PATH\n", bin)
	}
	return fmt.Errorf("%s: %s is not on PATH", what, bin)
}
