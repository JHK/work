package sessions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Claude Code's mangling, reproduced: every non-alphanumeric becomes a dash,
// one per UTF-16 code unit, and a name past bucketLen is truncated and hashed.
func TestBucket(t *testing.T) {
	tests := []struct{ dir, want string }{
		{"/home/u/Code/repo/.worktrees/bd-42", "-home-u-Code-repo--worktrees-bd-42"},
		{"/home/u/Code/my_repo/.worktrees/bd-42", "-home-u-Code-my-repo--worktrees-bd-42"},
		{"/home/u/Code/some repo (v2)/.worktrees/x+y@z", "-home-u-Code-some-repo--v2---worktrees-x-y-z"},
		{"/home/u/Code/repo-\U0001F600/.worktrees/bd-1", "-home-u-Code-repo-----worktrees-bd-1"},
		{"/home/u/" + strings.Repeat("a", 220) + "/wt", "-home-u-" + strings.Repeat("a", 192) + "-ekpdbl"},
	}
	for _, tt := range tests {
		if got := bucket("/home/u", tt.dir); got != filepath.Join("/home/u/.claude/projects", tt.want) {
			t.Errorf("bucket(%q) = %q, want the bucket %q", tt.dir, got, tt.want)
		}
	}
}

func TestListMissingBucket(t *testing.T) {
	got, err := Claude{Home: t.TempDir()}.List("/nowhere")
	if err != nil || got != nil {
		t.Errorf("List() = %v, %v; want no sessions and no error", got, err)
	}
}

func TestListTitlesAndOrder(t *testing.T) {
	home := t.TempDir()
	dir := "/w/t"

	write(t, home, dir, "custom", []string{
		`{"type":"ai-title","aiTitle":"model's guess"}`,
		`{"type":"custom-title","customTitle":"deliberate name"}`,
		`{"type":"last-prompt","lastPrompt":"a prompt"}`,
	}, time.Unix(300, 0))

	write(t, home, dir, "ai", []string{
		`{"type":"ai-title","aiTitle":"stale"}`,
		`{"type":"last-prompt","lastPrompt":"a prompt"}`,
		`{"type":"ai-title","aiTitle":"latest"}`,
	}, time.Unix(200, 0))

	write(t, home, dir, "prompt", []string{
		`{"type":"user","message":"not a title event"}`,
		`{"type":"last-prompt","lastPrompt":"` + strings.Repeat("x", 100) + `"}`,
	}, time.Unix(100, 0))

	write(t, home, dir, "bare", []string{`{"type":"user"}`}, time.Unix(50, 0))

	got, err := Claude{Home: home}.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []Session{
		{ID: "custom", Title: "deliberate name"},
		{ID: "ai", Title: "latest"},
		{ID: "prompt", Title: strings.Repeat("x", promptTitleLen)},
		{ID: "bare", Title: "(untitled)"},
	}
	if len(got) != len(want) {
		t.Fatalf("List() returned %d sessions, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].ID != want[i].ID || got[i].Title != want[i].Title {
			t.Errorf("session %d = %s/%q, want %s/%q", i, got[i].ID, got[i].Title, want[i].ID, want[i].Title)
		}
	}
}

// A transcript's message bodies dwarf its title events; skipping them must not
// cost the events that follow.
func TestListSkipsOversizedLines(t *testing.T) {
	home := t.TempDir()
	dir := "/w/t"
	write(t, home, dir, "big", []string{
		`{"type":"assistant","text":"` + strings.Repeat("x", 2*maxLine) + `"}`,
		`{"type":"ai-title","aiTitle":"found anyway"}`,
	}, time.Unix(100, 0))

	got, err := Claude{Home: home}.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "found anyway" {
		t.Errorf("List() = %+v, want the title after the oversized line", got)
	}
}

func write(t *testing.T, home, dir, id string, lines []string, mod time.Time) {
	t.Helper()
	b := bucket(home, dir)
	if err := os.MkdirAll(b, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(b, id+".jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mod, mod); err != nil {
		t.Fatal(err)
	}
}
