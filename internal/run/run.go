// Package run spawns the command-line tools the adapters read their state from.
package run

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// gone are the commands this run has already found missing.
var gone sync.Map

// Output runs bin in dir and returns its stdout, trimmed of the trailing
// newline. A failing tool's first line of stderr becomes the message. The
// command as it was run is said at info, which --log-level=info puts on stderr.
//
// A tool the machine does not have is out for the rest of the run: the first
// question to it fails and every later one fails with it, unasked.
func Output(dir, bin string, args ...string) (string, error) {
	return output(dir, false, bin, args...)
}

// InEnglish runs bin the way [Output] does, with the tool's own messages left
// untranslated, for a caller that reads what came back rather than only handing
// it on.
func InEnglish(dir, bin string, args ...string) (string, error) {
	return output(dir, true, bin, args...)
}

func output(dir string, english bool, bin string, args ...string) (string, error) {
	what := CommandLine(bin, args...)
	if _, missing := gone.Load(bin); missing {
		return "", absent(what, bin)
	}

	cmd := Command(dir, bin, args...)
	if english {
		cmd.Env = append(os.Environ(), "LC_ALL=C")
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			gone.Store(bin, true)
			return "", absent(what, bin)
		}
		if msg, _, _ := strings.Cut(strings.TrimSpace(stderr.String()), "\n"); msg != "" {
			return "", fmt.Errorf("%s: %s", what, msg)
		}
		return "", fmt.Errorf("%s: %w", what, err)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// Command is bin as [Output] would run it, said the same way, for a caller that
// has to reach the process itself: one that writes its stdin, or one that reads
// its exit status apart.
func Command(dir, bin string, args ...string) *exec.Cmd {
	slog.Info(CommandLine(bin, args...))
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	return cmd
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
		return v, fmt.Errorf("%s: %w", CommandLine(bin, args...), err)
	}
	return v, nil
}

// CommandLine is bin and its arguments as a shell would spell them, for whoever
// has to run the same thing by hand.
func CommandLine(bin string, args ...string) string {
	if len(args) == 0 {
		return bin
	}
	return bin + " " + strings.Join(args, " ")
}

// absent is what a tool the machine does not have refuses with.
func absent(what, bin string) error {
	return fmt.Errorf("%s: %s is not on PATH", what, bin)
}

// Forget drops what this run has found missing, so a tool that has since arrived
// on PATH is asked again. It is for tests, whose machine changes between cases.
func Forget() { gone.Clear() }
