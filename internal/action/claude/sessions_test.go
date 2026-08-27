package claude

// The transcript reader is what no command reaches: one sees only which
// conversation --resume was handed, and the mangling, the ordering and how far
// into an event the reader goes decide that between them.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JHK/work-cli/internal/testenv"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// transcript is one conversation as it sits on disk.
type transcript struct {
	id    string
	lines []string
	mod   time.Time
}

// record takes the case a home of its own and files the transcripts under the
// bucket Claude Code would file them in. None leaves the bucket itself unmade,
// which is a directory never worked in.
func record(t *testing.T, dir string, held ...transcript) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, c := range held {
		path := filepath.Join(Bucket(home, dir), c.id+".jsonl")
		testenv.Write(t, path, strings.Join(c.lines, "\n")+"\n")
		require.NoError(t, os.Chtimes(path, c.mod, c.mod))
	}
}

// Claude Code's mangling, reproduced: every non-alphanumeric becomes a dash, one
// per UTF-16 code unit, and a name past bucketLen is truncated and hashed.
func TestBucket(t *testing.T) {
	tests := []struct{ name, dir, want string }{
		{"a plain path", "/home/u/Code/repo/.worktrees/bd-42", "-home-u-Code-repo--worktrees-bd-42"},
		{"an underscore", "/home/u/Code/my_repo/.worktrees/bd-42", "-home-u-Code-my-repo--worktrees-bd-42"},
		{"spaces and punctuation", "/home/u/Code/some repo (v2)/.worktrees/x+y@z", "-home-u-Code-some-repo--v2---worktrees-x-y-z"},
		{"a rune of two code units", "/home/u/Code/repo-\U0001F600/.worktrees/bd-1", "-home-u-Code-repo-----worktrees-bd-1"},
		{"a name past bucketLen", "/home/u/" + strings.Repeat("a", 220) + "/wt", "-home-u-" + strings.Repeat("a", 192) + "-ekpdbl"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := filepath.Join("/home/u/.claude/projects", tt.want)
			require.Equalf(t, want, Bucket("/home/u", tt.dir), "the bucket %q is filed under", tt.dir)
		})
	}
}

// Only what the agent would return to is on offer, newest first, against the
// invocations docs/references/claude.md tabulates.
func TestCarriedOffersTheConversationsWorthReturningTo(t *testing.T) {
	const dir = "/w/t"
	tests := []struct {
		name string
		has  []transcript
		want []string
	}{
		{
			// Nothing recorded, so the bucket was never made: a missing one is not an error.
			"a directory never worked in carries none",
			nil,
			nil,
		},
		{
			"newest first",
			[]transcript{
				{"oldest", []string{`{"type":"user"}`}, time.Unix(100, 0)},
				{"newest", []string{`{"type":"user"}`}, time.Unix(300, 0)},
				{"middle", []string{`{"type":"user"}`}, time.Unix(200, 0)},
			},
			[]string{"newest", "middle", "oldest"},
		},
		{
			// The second transcript quotes the key in a message body, where it is text
			// rather than that transcript's entrypoint.
			"print mode is not on offer",
			[]transcript{
				{"printed", []string{
					`{"type":"system","entrypoint":"sdk-cli"}`,
					`{"type":"user","message":"summarise this"}`,
				}, time.Unix(200, 0)},
				{"interactive", []string{
					`{"type":"user","entrypoint":"cli","message":"what is \"entrypoint\":\"sdk-cli\"?"}`,
				}, time.Unix(100, 0)},
			},
			[]string{"interactive"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record(t, dir, tt.has...)

			got, err := carried(dir)
			require.NoError(t, err)
			testenv.Equal(t, tt.want, got, "the wrong conversations are on offer")
		})
	}
}

// The entrypoint is read off a whole event within a bounded prefix of the file: a
// line headLen severs is a fragment, and one the writer left unterminated is not.
func TestPrintModeReadsAWholeEventWithinABoundedHead(t *testing.T) {
	unnamed := `{"type":"user","message":"nothing named here"}` + "\n"
	tests := []struct {
		name string
		body string
		want bool
	}{
		{
			"a transcript naming no entrypoint",
			strings.Repeat(unnamed, 1+headLen/len(unnamed)),
			false,
		},
		{
			// Padded with spaces, so the severed prefix parses and only the cut decides.
			"an event the head severed",
			`{"type":"system","entrypoint":"sdk-cli"}` + strings.Repeat(" ", headLen) + "\n",
			false,
		},
		{
			"a final event left unterminated",
			`{"type":"system","entrypoint":"sdk-cli"}`,
			true,
		},
		{
			// The oversized event names the key below the top level, where it decides
			// nothing, and the event behind it still settles the transcript.
			"an event too long to read hides nothing behind it",
			`{"type":"user","message":{"entrypoint":"sdk-cli","pad":"` + strings.Repeat("x", headLen/4) + `"}}` + "\n" +
				`{"type":"system","entrypoint":"sdk-cli"}` + "\n",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.body)

			require.Equal(t, tt.want, printMode(r))
			require.LessOrEqual(t, r.Size()-int64(r.Len()), int64(headLen))
		})
	}
}
