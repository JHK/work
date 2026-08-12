package claude

// Reading which conversations a worktree carries rests on undocumented Claude
// Code internals: the path mangling behind [bucket], and the entrypoint its JSONL
// transcript events carry.

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

// transcripts are the conversations a worktree carries, newest first. Only what
// the agent would return to counts, so whatever its own picker hides is left out.
type transcripts interface {
	list(dir string) ([]string, error)
}

// recorded reads the transcripts Claude Code writes under home.
type recorded struct {
	home string // user home directory; empty means os.UserHomeDir
}

// list reports the conversations recorded for the given working directory, most
// recently touched first. A directory that was never worked in has none, which is
// not an error.
func (c recorded) list(dir string) ([]string, error) {
	home := c.home
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

	type file struct {
		name string
		mod  time.Time
	}
	var files []file
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".jsonl" {
			continue
		}
		info, err := e.Info()
		if err != nil || !listed(filepath.Join(b, e.Name())) {
			continue
		}
		files = append(files, file{name: e.Name(), mod: info.ModTime()})
	}
	// Stable, so that transcripts sharing a timestamp do not reorder between runs.
	slices.SortStableFunc(files, func(a, b file) int { return b.mod.Compare(a.mod) })

	var out []string
	for _, f := range files {
		out = append(out, strings.TrimSuffix(f.name, ".jsonl"))
	}
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

type event struct {
	Entrypoint string `json:"entrypoint"`
}

// listed reports whether a file is a conversation to return to: one that cannot
// be read, or that `claude -p` wrote, is not.
func listed(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	return !printMode(f)
}

// printEntrypoint marks a transcript `claude -p` wrote.
const printEntrypoint = "sdk-cli"

// headLen is how much of a transcript's start is worth reading: every message
// event carries the entrypoint, and the metadata ahead of the first is kilobytes.
const headLen = 256 << 10

// printMode reports whether the transcript came from print mode, which is the
// entrypoint its events name. A line headLen severs is a fragment, not an event.
func printMode(r io.Reader) bool {
	head := &io.LimitedReader{R: r, N: headLen}
	br := bufio.NewReader(head)
	for {
		line, err := br.ReadString('\n')
		// Short of a newline: a whole event only where the file, not the budget, ran out.
		if err != nil && (head.N == 0 || line == "") {
			return false
		}
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
}
