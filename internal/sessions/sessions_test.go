package sessions

import (
	"os"
	"path/filepath"
	"slices"
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

func TestListOrder(t *testing.T) {
	home := t.TempDir()
	dir := "/w/t"

	write(t, home, dir, "oldest", []string{`{"type":"user"}`}, time.Unix(100, 0))
	write(t, home, dir, "newest", []string{`{"type":"user"}`}, time.Unix(300, 0))
	write(t, home, dir, "middle", []string{`{"type":"user"}`}, time.Unix(200, 0))

	got, err := Claude{Home: home}.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []Session{{ID: "newest"}, {ID: "middle"}, {ID: "oldest"}}
	if !slices.Equal(got, want) {
		t.Errorf("List() = %+v, want %+v", got, want)
	}
}

// claude -p writes a transcript like any other, and neither --continue nor the
// picker offers one, so a worktree holding nothing else holds no conversation.
func TestListHidesPrintMode(t *testing.T) {
	home := t.TempDir()
	dir := "/w/t"

	write(t, home, dir, "printed", []string{
		`{"type":"system","entrypoint":"sdk-cli"}`,
		`{"type":"user","message":"summarise this"}`,
	}, time.Unix(200, 0))

	// The key inside a message body is text, not this transcript's entrypoint.
	write(t, home, dir, "interactive", []string{
		`{"type":"user","entrypoint":"cli","message":"what is \"entrypoint\":\"sdk-cli\"?"}`,
	}, time.Unix(100, 0))

	got, err := Claude{Home: home}.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "interactive" {
		t.Errorf("List() = %+v, want the interactive transcript alone", got)
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
