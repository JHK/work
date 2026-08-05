package work

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	const repo = "/repo"
	tests := []struct {
		arg  string
		kind Kind
		id   string
		name string
	}{
		{"bd-42", KindBead, "bd-42", "bd-42"},
		{"7", KindPR, "7", "pr-7"},
		{"https://github.com/o/r/pull/91", KindPR, "91", "pr-91"},
		{"https://github.com/o/r/pull/91/files", KindPR, "91", "pr-91"},
		{"pull-request-1", KindBead, "pull-request-1", "pull-request-1"},
		{"007", KindPR, "7", "pr-7"},
	}
	for _, tt := range tests {
		got, err := Resolve(repo, tt.arg)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", tt.arg, err)
		}
		if got.Kind != tt.kind || got.ID != tt.id || got.Name != tt.name {
			t.Errorf("Resolve(%q) = %+v, want kind %v id %q name %q", tt.arg, got, tt.kind, tt.id, tt.name)
		}
		if want := filepath.Join(repo, ".worktrees", tt.name); got.Path != want {
			t.Errorf("Resolve(%q).Path = %q, want %q", tt.arg, got.Path, want)
		}
	}

	// An identifier becomes a directory name and a refspec; anything that would
	// leave .worktrees, or that git would reject, has to be refused up front.
	for _, arg := range []string{"", "..", "../../etc", "a/b", "/", ".", "-5", "--yes", "docs/pull/99-notes.md"} {
		if got, err := Resolve(repo, arg); err == nil {
			t.Errorf("Resolve(%q) = %+v, want an error", arg, got)
		}
	}
}

// TargetAt inverts Resolve's naming, so a worktree the picker offers and the
// same target named on the command line have to agree.
func TestTargetAt(t *testing.T) {
	const repo = "/repo"
	for _, arg := range []string{"bd-42", "7", "pr-91"} {
		want, err := Resolve(repo, arg)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", arg, err)
		}
		got, ok := TargetAt(repo, want.Path)
		if !ok || got != want {
			t.Errorf("TargetAt(%q) = %+v, %v; want %+v", want.Path, got, ok, want)
		}
	}

	// A numeric bead id read back off a worktree stays a bead: the PR heuristic
	// belongs to command-line input, not to a directory that already exists.
	beadDir := filepath.Join(repo, ".worktrees", "1234")
	if got, ok := TargetAt(repo, beadDir); !ok || got.Kind != KindBead || got.ID != "1234" {
		t.Errorf("TargetAt(%q) = %+v, %v; want a bead", beadDir, got, ok)
	}

	for _, path := range []string{
		filepath.Join(repo, "src"),                    // not under .worktrees
		filepath.Join(repo, ".worktrees", "a", "b"),   // nested too deep
		filepath.Join("/other", ".worktrees", "bd-1"), // another repo
		filepath.Join(repo, ".worktrees", "-dash"),    // unusable as an argument
	} {
		if got, ok := TargetAt(repo, path); ok {
			t.Errorf("TargetAt(%q) = %+v, want no target", path, got)
		}
	}
}

func TestSlug(t *testing.T) {
	tests := []struct{ title, want string }{
		{"Port work to Go", "port-work-to-go"},
		{"Rate-limit /api/upload", "rate-limit-api-upload"},
		{"  Trailing punctuation!!  ", "trailing-punctuation"},
		{strings.Repeat("ab ", 30), strings.Repeat("ab-", 13) + "a"},
		{"—", ""},
	}
	for _, tt := range tests {
		if got := slug(tt.title); got != tt.want {
			t.Errorf("slug(%q) = %q, want %q", tt.title, got, tt.want)
		}
		if len(slug(tt.title)) > slugLen {
			t.Errorf("slug(%q) is longer than %d", tt.title, slugLen)
		}
	}
}

func TestBranch(t *testing.T) {
	pr, _ := Resolve("/repo", "7")
	if got, err := (State{Target: pr}).Branch(); err != nil || got != "pr-7" {
		t.Errorf("PR branch = %q, %v", got, err)
	}

	bead, _ := Resolve("/repo", "bd-42")
	s := State{Target: bead}
	s.Bead.Title = "Port work to Go"
	if got, err := s.Branch(); err != nil || got != "bd-42-port-work-to-go" {
		t.Errorf("bead branch = %q, %v", got, err)
	}
}
