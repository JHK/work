// Package run spawns the command-line tools the adapters read their state from.
package run

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Output runs bin in dir and returns its stdout, trimmed of the trailing
// newline. A tool that fails diagnoses itself better than its exit status does,
// so its first line of stderr becomes the message.
func Output(dir, bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		what := bin
		if len(args) > 0 {
			what += " " + args[0]
		}
		if msg, _, _ := strings.Cut(strings.TrimSpace(stderr.String()), "\n"); msg != "" {
			return "", fmt.Errorf("%s: %s", what, msg)
		}
		return "", fmt.Errorf("%s: %w", what, err)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}
