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
// newline. A failing tool's first line of stderr becomes the message.
//
// A tool the machine does not have is out for the rest of the run: the first
// question to it fails and every later one fails with it, unasked.
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

// asked is the tool and the question put to it, as a refusal names them.
func asked(bin string, args []string) string {
	if len(args) == 0 {
		return bin
	}
	return bin + " " + args[0]
}

// absent is a tool work reached for and the machine does not have. It warns once
// here as well as refusing: a caller that swallows the error would go quiet.
func absent(what, bin string) error {
	if _, before := gone.LoadOrStore(bin, true); !before {
		_, _ = fmt.Fprintf(warnings, "work: %s is not on PATH\n", bin)
	}
	return fmt.Errorf("%s: %s is not on PATH", what, bin)
}
