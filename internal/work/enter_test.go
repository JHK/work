package work

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/beads"
	"github.com/JHK/work-cli/internal/config"
)

// The vetting guards the claim, so it runs wherever a ticket's worktree is
// about to be created, whatever the invocation opens on and whether or not it
// will claim. The repository here is not one, so anything reaching the
// provisioning fails rather than making a worktree.
func TestEnterVetsEveryWay(t *testing.T) {
	t.Setenv("VISUAL", "vi")
	e := Env{Repo: filepath.Join(t.TempDir(), "not-a-repo"), Config: config.Default()}
	s := State{
		Target: Target{Kind: KindBead, ID: "bd-1", Name: "bd-1"},
		Bead:   beads.Bead{ID: "bd-1", Status: "closed", Type: "task", AcceptanceCriteria: "It works"},
	}

	tests := []struct {
		name string
		opts Options
	}{
		{"a session", Options{}},
		{"a shell", Options{Shell: true}},
		{"an editor", Options{Editor: true}},
		{"no claim", Options{NoClaim: true}},
		{"a shell and no claim", Options{Shell: true, NoClaim: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := e.enter(s, false, tt.opts)
			if err == nil || !strings.Contains(err.Error(), "already closed") {
				t.Errorf("enter(%+v) = %v; want the closed bead refused, saying which rule it broke", tt.opts, err)
			}
		})
	}
}

// A worktree that already exists is re-entered whatever the flags say about
// claiming: there is nothing to vet, nothing to create and nothing to claim.
func TestEnterOpensOn(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/fish")
	t.Setenv("VISUAL", "vi")
	e := Env{Repo: filepath.Join(t.TempDir(), "not-a-repo"), Config: config.Default()}
	s := State{Target: Target{Kind: KindBead, ID: "bd-1", Name: "bd-1"}, Path: "/wt", Exists: true}

	tests := []struct {
		name string
		opts Options
		want []string
	}{
		{"a shell by default", Options{}, []string{"/usr/bin/fish"}},
		{"asked for outright", Options{Shell: true}, []string{"/usr/bin/fish"}},
		{"no claim changes nothing", Options{NoClaim: true}, []string{"/usr/bin/fish"}},
		{"an editor", Options{Editor: true}, []string{"vi", "/wt"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := e.enter(s, false, tt.opts)
			if err != nil {
				t.Fatalf("enter(%+v): %v", tt.opts, err)
			}
			if !slices.Equal(got.Handoff.Run, tt.want) {
				t.Errorf("enter(%+v) runs %q; want %q", tt.opts, got.Handoff.Run, tt.want)
			}
			if want := !tt.opts.Editor; got.Shell != want {
				t.Errorf("enter(%+v).Shell = %v; want %v", tt.opts, got.Shell, want)
			}
		})
	}
}
