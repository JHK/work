package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

// nothingInstalled stands the shell on a PATH holding git alone, which is a
// machine none of the tools work reaches were ever installed on. The stand-ins
// go with it, so what a run asks for reaches nothing.
func nothingInstalled(t *testing.T) {
	t.Helper()
	git, err := exec.LookPath("git")
	require.NoError(t, err, "git is not on PATH to begin with")
	dir := t.TempDir()
	// A file rather than a symlink, which is what every other stand-in here is.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "git"),
		[]byte("#!/bin/sh\nexec "+git+" \"$@\"\n"), 0o755), "put git on a PATH of the case's own")
	t.Setenv("PATH", dir)
}

// A tool the machine does not have is refused as the question that reached for
// it, naming the question and the tool that is not there: internal/run.
func TestAToolTheMachineDoesNotHaveIsRefused(t *testing.T) {
	s := repository(t)
	s.settings(on("beads"))
	nothingInstalled(t)

	r := s.run("go", "bd-1")

	r.came(t, result{Code: 1, Errored: []string{listed + ": bd is not on PATH"}})
}

// An answer that is not the JSON work asked for is a refusal like any other,
// naming the command as it was run so a reader sees what answered instead.
func TestAnAnswerThatIsNotJSONNamesTheCommandThatGaveIt(t *testing.T) {
	s := repository(t, testenv.Stub{Name: "bd", Says: "not json"})
	s.settings(on("beads"))

	r := s.run("go", "bd-1")

	r.came(t, result{Code: 1, Asked: []string{listed}}, apart)
	r.saying(t, listed+": ")
}
