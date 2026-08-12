package claude

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
	got, err := recorded{home: t.TempDir()}.list("/nowhere")
	if err != nil || got != nil {
		t.Errorf("list() = %v, %v; want no conversations and no error", got, err)
	}
}

func TestListOrder(t *testing.T) {
	home := t.TempDir()
	dir := "/w/t"

	write(t, home, dir, "oldest", []string{`{"type":"user"}`}, time.Unix(100, 0))
	write(t, home, dir, "newest", []string{`{"type":"user"}`}, time.Unix(300, 0))
	write(t, home, dir, "middle", []string{`{"type":"user"}`}, time.Unix(200, 0))

	got, err := recorded{home: home}.list(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"newest", "middle", "oldest"}
	if !slices.Equal(got, want) {
		t.Errorf("list() = %q, want %q", got, want)
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

	got, err := recorded{home: home}.list(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "interactive" {
		t.Errorf("list() = %q, want the interactive transcript alone", got)
	}
}

// An event too long to be one of the short ones read here must not hide the
// events behind it.
func TestListHidesPrintModeBehindAnOversizedLine(t *testing.T) {
	home := t.TempDir()
	dir := "/w/t"

	// The oversized event names the key below the top level, where it decides nothing.
	write(t, home, dir, "printed", []string{
		`{"type":"user","message":{"entrypoint":"sdk-cli","pad":"` + strings.Repeat("x", headLen/4) + `"}}`,
		`{"type":"system","entrypoint":"sdk-cli"}`,
	}, time.Unix(100, 0))
	write(t, home, dir, "interactive", []string{`{"type":"user","entrypoint":"cli"}`}, time.Unix(200, 0))

	got, err := recorded{home: home}.list(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "interactive" {
		t.Errorf("list() = %q, want the interactive transcript alone", got)
	}
}

// A transcript naming no entrypoint stays on offer, and costs a bounded prefix
// rather than the whole file.
func TestPrintModeStopsAtHeadLen(t *testing.T) {
	line := `{"type":"user","message":"nothing named here"}` + "\n"
	r := strings.NewReader(strings.Repeat(line, 1+headLen/len(line)))

	if printMode(r) {
		t.Error("printMode() = true, want false for a transcript naming no entrypoint")
	}
	if read := r.Size() - int64(r.Len()); read > headLen {
		t.Errorf("printMode read %d bytes, want at most headLen (%d)", read, headLen)
	}
}

// The line headLen severs is a fragment, whatever it appears to name. Padded
// with spaces, so the severed prefix parses and only the cut decides.
func TestPrintModeIgnoresASeveredLine(t *testing.T) {
	severed := `{"type":"system","entrypoint":"sdk-cli"}` + strings.Repeat(" ", headLen) + "\n"
	if printMode(strings.NewReader(severed)) {
		t.Error("printMode() = true, want false: the line was severed, not read")
	}
}

// A writer that stopped short of the newline still wrote a whole event.
func TestPrintModeReadsAnUnterminatedFinalLine(t *testing.T) {
	if !printMode(strings.NewReader(`{"type":"system","entrypoint":"sdk-cli"}`)) {
		t.Error("printMode() = false, want true: the file ran out, not the budget")
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
