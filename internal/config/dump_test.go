package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JHK/work-cli/internal/testenv"
)

// A setting the dump leaves out is one nothing would ever show, and a key
// spelled otherwise than a settings file spells it is one nothing would load
// back. Both are read off the tables themselves rather than restated here.
func TestDumpNamesEverySetting(t *testing.T) {
	var want []string
	for table := range reflect.TypeFor[Config]().Fields() {
		for field := range table.Type.Fields() {
			want = append(want, strings.ToLower(table.Name)+"."+spelling(field))
		}
	}

	var got []string
	for _, k := range Default().keys() {
		got = append(got, k.name)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the dump names %q; the tables carry %q", got, want)
	}
}

// spelling is the key a field is written as: its toml tag, or its name
// lowercased where it has none, which is what the decoder matches.
func spelling(f reflect.StructField) string {
	if tag, ok := f.Tag.Lookup("toml"); ok {
		return tag
	}
	return strings.ToLower(f.Name)
}

// Every key names the layer that set it: the file, or the compiled-in default
// where no file did. The repository's file wins where both set one, and is named
// as it is written rather than by its path.
func TestDumpNamesTheLayerBehindEachKey(t *testing.T) {
	repo, home := t.TempDir(), testenv.Home(t)
	user := filepath.Join(home, userRelPath)
	testenv.Write(t, user, directory("trees")+"[open]\nshell = [\"fish\"]\n[mise]\nenabled = true\n")
	testenv.Write(t, filepath.Join(repo, repoFile), "[open]\nshell = [\"zsh\"]\n[action]\nenter = \"shell\"\n[beads]\nenabled = true\n")

	got, err := Dump(repo)
	if err != nil {
		t.Fatalf("Dump: %v", err)
	}
	from := sourced(got)
	// A system says which layer switched it on, and one nothing named says it is
	// off and where that comes from, so what a dump shows is every system's state.
	want := map[string]string{
		"worktree.directory": user,
		"open.shell":         repoFile,
		"action.enter":       repoFile,
		"action.create":      compiledIn,
		"branch.ticket":      compiledIn,
		"beads.enabled":      repoFile,
		"mise.enabled":       user,
		"claude.enabled":     compiledIn,
	}
	for key, layer := range want {
		if from[key] != layer {
			t.Errorf("%s came from %q; want %q", key, from[key], layer)
		}
	}
}

// What the dump prints is a settings file: loading it back names the same
// configuration, whatever layer each key came from.
func TestDumpLoadsBack(t *testing.T) {
	repo, home := t.TempDir(), testenv.Home(t)
	testenv.Write(t, filepath.Join(home, userRelPath), directory("trees")+branch("{{.ID}}", "review/{{.Number}}"))
	// A quote and a tab survive the printing, being written as TOML escapes.
	testenv.Write(t, filepath.Join(repo, repoFile),
		"[action]\nenter = \"diff\"\n[claude]\nstart-session = [\"claude\", \"--name=\\\"{{.Name}}\\\"\", \"a\\tb\"]\n")

	text, err := Dump(repo)
	if err != nil {
		t.Fatalf("Dump: %v", err)
	}

	// A second repository whose whole configuration is that text.
	back := t.TempDir()
	testenv.Write(t, filepath.Join(back, repoFile), text)
	printed, _, err := merge(repo)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	loaded, _, err := merge(back)
	if err != nil {
		t.Fatalf("merge of the dump: %v", err)
	}
	if !reflect.DeepEqual(printed.keys(), loaded.keys()) {
		t.Errorf("the dump loads back as %+v; want %+v", loaded.keys(), printed.keys())
	}
}

// A configuration work would refuse to load is refused here too, whichever
// layer carries the value, and nothing is printed of it.
func TestDumpRefusesWhatLoadRefuses(t *testing.T) {
	tests := []struct {
		name       string
		user, repo string
	}{
		{"the repository's file", "", "[action]\nenter = \"vim\"\n"},
		{"the user's file", "[open]\ndiff = []\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, home := t.TempDir(), testenv.Home(t)
			if tt.user != "" {
				testenv.Write(t, filepath.Join(home, userRelPath), tt.user)
			}
			if tt.repo != "" {
				testenv.Write(t, filepath.Join(repo, repoFile), tt.repo)
			}

			got, err := Dump(repo)
			if err == nil {
				t.Fatalf("Dump() = %q; want a refusal", got)
			}
			if got != "" {
				t.Errorf("Dump() printed %q; want nothing", got)
			}
		})
	}
}

// sourced reads a dump back as the layer named above each key, under the dotted
// name a settings file would spell: a leaf alone would not be unique, `enabled`
// standing in every system's table.
func sourced(text string) map[string]string {
	from := map[string]string{}
	table, comment := "", ""
	for line := range strings.SplitSeq(text, "\n") {
		if name, ok := strings.CutPrefix(line, "["); ok {
			table = strings.TrimSuffix(name, "]")
			continue
		}
		if after, ok := strings.CutPrefix(line, "# "); ok {
			comment = after
			continue
		}
		if key, _, ok := strings.Cut(line, " = "); ok {
			from[table+"."+key] = comment
		}
	}
	return from
}
