// Package sessions defines what work asks an agent about a worktree, and
// answers it for Claude Code. [Claude] rests entirely on undocumented
// internals: the path-mangling scheme behind [bucket], and the event types
// inside the JSONL transcripts.
package sessions

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// Session is one resumable transcript.
type Session struct {
	ID       string
	Title    string
	Modified time.Time
}

// Conversations is what work needs of an agent beyond a command to run: which
// conversations a worktree carries, newest first. Only what the agent would
// return to counts, so an implementation leaves out whatever its own picker
// hides.
type Conversations interface {
	List(dir string) ([]Session, error)
}

// Claude reads the transcripts Claude Code writes under Home.
type Claude struct {
	Home string // user home directory; empty means os.UserHomeDir
}

// List reports the sessions recorded for the given working directory, most
// recently touched first. A directory that was never worked in has none, which
// is not an error.
func (c Claude) List(dir string) ([]Session, error) {
	home := c.Home
	if home == "" {
		var err error
		if home, err = os.UserHomeDir(); err != nil {
			return nil, err
		}
	}

	b := bucket(home, dir)
	entries, err := os.ReadDir(b)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var out []Session
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		title, listed := transcript(filepath.Join(b, e.Name()), info.Size())
		if !listed {
			continue
		}
		out = append(out, Session{
			ID:       strings.TrimSuffix(e.Name(), ".jsonl"),
			Title:    title,
			Modified: info.ModTime(),
		})
	}
	// Stable, so that transcripts sharing a timestamp do not reorder between runs.
	slices.SortStableFunc(out, func(a, b Session) int { return b.Modified.Compare(a.Modified) })
	return out, nil
}

// bucketLen is the length past which Claude Code truncates a mangled path and
// appends a hash of the original.
const bucketLen = 200

// bucket is the directory Claude Code files a working directory's transcripts
// under: the absolute path with every non-alphanumeric character flattened to a
// dash, truncated with a hash suffix when that runs long.
func bucket(home, dir string) string {
	var b strings.Builder
	b.Grow(len(dir))
	for _, r := range dir {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r > 0xFFFF:
			// Two dashes: the mangler runs over a JS string, one per code unit.
			b.WriteString("--")
		default:
			b.WriteByte('-')
		}
	}
	name := b.String()
	if len(name) > bucketLen {
		name = name[:bucketLen] + "-" + strconv.FormatInt(pathHash(dir), 36)
	}
	return filepath.Join(home, ".claude", "projects", name)
}

// pathHash reproduces the 32-bit string hash Claude Code suffixes a truncated
// bucket with: h = h*31 + unit, over UTF-16 code units, made positive.
func pathHash(s string) int64 {
	var h int32
	for _, u := range utf16.Encode([]rune(s)) {
		h = h*31 + int32(u)
	}
	if h < 0 {
		return -int64(h)
	}
	return int64(h)
}

// promptTitleLen caps how much of a bare prompt stands in for a title.
const promptTitleLen = 60

type event struct {
	Type        string `json:"type"`
	CustomTitle string `json:"customTitle"`
	AITitle     string `json:"aiTitle"`
	LastPrompt  string `json:"lastPrompt"`
	Entrypoint  string `json:"entrypoint"`
}

// tailLen is how much of a transcript's end is worth reading. Title events are
// rewritten as the session runs, so the last ones sit within a few kilobytes of
// the end while the bulk of the file is message bodies.
const tailLen = 256 << 10

// transcript reads a file's title and whether it is a conversation to return to
// at all: one that cannot be read, or that `claude -p` wrote, is not.
func transcript(path string, size int64) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer func() { _ = f.Close() }()

	if printMode(io.NewSectionReader(f, 0, headLen)) {
		return "", false
	}
	return title(f, size), true
}

// printEntrypoint marks a transcript `claude -p` wrote. The picker does not
// offer one, so a worktree carrying nothing else carries nothing.
const printEntrypoint = "sdk-cli"

// headLen is how much of a transcript's start is worth reading. Every message
// event carries the entrypoint, and the metadata ahead of the first one runs to
// a few kilobytes.
const headLen = 64 << 10

// printMode reports whether the transcript came from print mode, which is the
// entrypoint its events name.
func printMode(r io.Reader) bool {
	for line := range lines(r) {
		// A message body may quote the key, so the parsed value decides, not the text.
		if !strings.Contains(line, `"entrypoint"`) {
			continue
		}
		var e event
		if json.Unmarshal([]byte(line), &e) != nil || e.Entrypoint == "" {
			continue
		}
		return e.Entrypoint == printEntrypoint
	}
	return false
}

// title picks the best name a transcript offers. A name from `claude --name` or
// /name is deliberate, so it outranks the model's own title; a bare prompt is
// the last resort.
func title(r io.ReaderAt, size int64) string {
	if t := titleIn(tail(r, size)); t != "" {
		return t
	}
	// Nothing in the tail: either the transcript is untitled, or one line spans
	// the whole window.
	if size > tailLen {
		if t := titleIn(io.NewSectionReader(r, 0, size)); t != "" {
			return t
		}
	}
	return "(untitled)"
}

// tail reads from the line boundary nearest the last tailLen bytes. Every read
// is positioned, so no other reader of the same file has to care about order.
func tail(r io.ReaderAt, size int64) io.Reader {
	if size <= tailLen {
		return io.NewSectionReader(r, 0, size)
	}
	br := bufio.NewReader(io.NewSectionReader(r, size-tailLen, tailLen))
	if _, err := br.ReadString('\n'); err != nil {
		// The window opened mid-line and never closed it.
		return strings.NewReader("")
	}
	return br
}

func titleIn(r io.Reader) string {
	var custom, ai, prompt string
	for line := range lines(r) {
		// Every line carries a type, so filter on the title events themselves.
		if !strings.Contains(line, `-title"`) && !strings.Contains(line, `"last-prompt"`) {
			continue
		}
		var e event
		if json.Unmarshal([]byte(line), &e) != nil {
			continue
		}
		switch e.Type {
		case "custom-title":
			custom = e.CustomTitle
		case "ai-title":
			ai = e.AITitle
		case "last-prompt":
			prompt = truncate(e.LastPrompt, promptTitleLen)
		}
	}

	for _, t := range []string{custom, ai, prompt} {
		if t = strings.TrimSpace(t); t != "" {
			return t
		}
	}
	return ""
}

// maxLine bounds what is worth parsing. Title events are short; the lines that
// blow past this are message bodies.
const maxLine = 64 << 10

func lines(r io.Reader) func(func(string) bool) {
	return func(yield func(string) bool) {
		br := bufio.NewReaderSize(r, maxLine)
		for {
			line, err := br.ReadSlice('\n')
			oversized := errors.Is(err, bufio.ErrBufferFull)
			for errors.Is(err, bufio.ErrBufferFull) {
				_, err = br.ReadSlice('\n')
			}
			if !oversized && len(line) > 0 && !yield(string(line)) {
				return
			}
			if err != nil {
				return
			}
		}
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
