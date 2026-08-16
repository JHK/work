// Package run spawns the command-line tools the adapters read their state from.
package run

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var (
	// Warnings is where what work says of the tools it reached for goes.
	Warnings io.Writer = os.Stderr

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
		return "", absent(asked(bin, args...), bin)
	}

	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		what := asked(bin, args...)
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

// JSON runs bin the way [Output] does and reads what it answered with into a
// value of type T. An answer that is not the JSON asked for is a refusal like
// any other, and names the command that gave it.
func JSON[T any](dir, bin string, args ...string) (T, error) {
	var v T
	out, err := Output(dir, bin, args...)
	if err != nil {
		return v, err
	}
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		return v, fmt.Errorf("%s: %w", asked(bin, args...), err)
	}
	return v, nil
}

// asked is the tool and the question put to it, as a refusal names them: the
// command as it was run, for whoever has to put it again by hand.
func asked(bin string, args ...string) string {
	if len(args) == 0 {
		return bin
	}
	return bin + " " + strings.Join(args, " ")
}

// absent is a tool work reached for and the machine does not have. The refusal
// is the answer to every question put to that tool for the rest of the run.
func absent(what, bin string) error {
	gone.Store(bin, true)
	return fmt.Errorf("%s: %s is not on PATH", what, bin)
}

// Forget drops what this run has found missing, so a tool that has since arrived
// on PATH is asked again. It is for tests, whose machine changes between cases.
func Forget() { gone.Clear() }

// Say puts a refusal on stderr, for the one caller that throws one away rather
// than handing it on. Everywhere else it is whoever is handed a refusal that
// says it, so nothing says one twice.
func Say(err error) {
	_, _ = fmt.Fprintln(Warnings, "work:", err)
}
