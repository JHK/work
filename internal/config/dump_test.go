package config

import (
	"reflect"
	"slices"
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

// A table carrying an enabled key is a system, and SystemNames is what spells
// them: one that list leaves out has no flag to reach it and no row in a dump.
func TestSystemNamesSpellsEveryTableThatCanBeSwitchedOn(t *testing.T) {
	var want []string
	for table := range reflect.TypeFor[Config]().Fields() {
		if _, ok := table.Type.FieldByName("Enabled"); ok {
			want = append(want, strings.ToLower(table.Name))
		}
	}

	slices.Sort(want)
	if got := slices.Sorted(slices.Values(SystemNames())); !slices.Equal(got, want) {
		t.Errorf("SystemNames spells %q; the tables that can be switched on are %q", got, want)
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

// Every key names where it came from: the file, or the compiled-in default where
// the file did not name it.
func TestDumpNamesTheLayerBehindEachKey(t *testing.T) {
	user := testenv.Settings(t, directory("trees")+
		"[claude]\nstart-session = [\"claude\"]\n[mise]\nenabled = true\n[action]\nenter = \"shell\"\n[beads]\nenabled = true\n")

	got, err := Dump()
	if err != nil {
		t.Fatalf("Dump: %v", err)
	}
	from := sourced(got)
	// A system says which layer switched it on, and one nothing named says it is
	// off and where that comes from, so what a dump shows is every system's state.
	want := map[string]string{
		"worktree.directory":   user,
		"claude.start-session": user,
		"action.enter":         user,
		"action.create":        compiledIn,
		"branch.ticket":        compiledIn,
		"beads.enabled":        user,
		"mise.enabled":         user,
		"claude.enabled":       compiledIn,
	}
	for key, layer := range want {
		if from[key] != layer {
			t.Errorf("%s came from %q; want %q", key, from[key], layer)
		}
	}
}

// What the dump prints is a settings file: loading it back names the same
// configuration, whether a key came from the file or from the defaults.
func TestDumpLoadsBack(t *testing.T) {
	// A quote and a tab survive the printing, being written as TOML escapes.
	testenv.Settings(t, directory("trees")+branch("{{.ID}}", "review/{{.Number}}")+
		"[action]\nenter = \"claude\"\n[claude]\nenabled = true\nstart-session = [\"claude\", \"--name=\\\"{{.Name}}\\\"\", \"a\\tb\"]\n")

	text, err := Dump()
	if err != nil {
		t.Fatalf("Dump: %v", err)
	}
	printed, _, err := read()
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	// A machine whose whole configuration is that text.
	testenv.Settings(t, text)
	loaded, _, err := read()
	if err != nil {
		t.Fatalf("read the dump back: %v", err)
	}
	if !reflect.DeepEqual(printed.keys(), loaded.keys()) {
		t.Errorf("the dump loads back as %+v; want %+v", loaded.keys(), printed.keys())
	}
}

// A configuration work would refuse to load is refused here too, and nothing is
// printed of it.
func TestDumpRefusesWhatLoadRefuses(t *testing.T) {
	tests := []struct{ name, body string }{
		{"an action nothing goes by", "[action]\nenter = \"vim\"\n"},
		{"a command that names nothing to run", "[claude]\nstart-session = []\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testenv.Settings(t, tt.body)

			got, err := Dump()
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
