package work

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// execHandoff is the environment variable the re-exec helper below keys on.
// Exec replaces the process, so the only way to observe it is from a child.
const execHandoff = "WORK_TEST_HANDOFF_DIR"

func TestMain(m *testing.M) {
	if dir := os.Getenv(execHandoff); dir != "" {
		if err := (Handoff{Dir: dir, Run: []string{"pwd"}}).Exec(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	os.Exit(m.Run())
}

func TestHandoffExec(t *testing.T) {
	dir := t.TempDir()

	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(), execHandoff+"="+dir)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("re-exec: %v", err)
	}
	// pwd resolves symlinks the temp dir may carry; compare what the shell sees.
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != want {
		t.Errorf("ran in %q, want %q", strings.TrimSpace(string(out)), want)
	}
}

func TestHandoffExecRejects(t *testing.T) {
	if err := (Handoff{Dir: "/tmp"}).Exec(); err == nil {
		t.Error("Exec with no command: want an error")
	}
	if err := (Handoff{Dir: "/no/such/dir", Run: []string{"pwd"}}).Exec(); err == nil {
		t.Error("Exec into a missing directory: want an error")
	}
	if err := (Handoff{Dir: "/tmp", Run: []string{"no-such-binary-xyz"}}).Exec(); err == nil {
		t.Error("Exec of a missing binary: want an error")
	}
}

func TestShell(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	if got := Shell(); len(got) != 1 || got[0] != "/usr/bin/fish" {
		t.Errorf("Shell() = %q, want the login shell", got)
	}

	t.Setenv("SHELL", "")
	if got := Shell(); len(got) != 1 || got[0] != "/bin/sh" {
		t.Errorf("Shell() = %q, want the fallback", got)
	}
}

func TestLaunchArgv(t *testing.T) {
	tests := []struct {
		name   string
		launch Launch
		want   []string
	}{
		{
			"fresh session",
			Launch{Name: "bd-1: a title", Prompt: "/start bd-1"},
			[]string{"claude", "--permission-mode", "auto", "--name", "bd-1: a title", "/start bd-1"},
		},
		{
			"model and effort",
			Launch{Prompt: "/start bd-1", Model: "opus", Effort: "high"},
			[]string{"claude", "--permission-mode", "auto", "--model", "opus", "--effort", "high", "/start bd-1"},
		},
		{
			"a resumed session takes no name",
			Launch{Resume: "abc", Name: "ignored"},
			[]string{"claude", "--permission-mode", "auto", "--resume", "abc"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.launch.Argv()
			if !slices.Equal(got, tt.want) {
				t.Errorf("Argv() = %q, want %q", got, tt.want)
			}
		})
	}
}
