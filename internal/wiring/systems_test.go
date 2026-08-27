package wiring

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/testenv"
	"github.com/JHK/work-cli/internal/work"
	"github.com/JHK/work-cli/internal/worktree"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) { testenv.Main(m) }

// A system wired off a name [config.SystemNames] leaves out is one no settings
// file reaches. Structural like R3 and R4: no command asserts an "every".
func TestTheSettingsSpellEverySystemTheWiringHas(t *testing.T) {
	repo := t.TempDir()
	// Read before the settings are written: each takes a settings home of its own.
	core := wired(Wire(repo, repo, load(t)))
	added := wired(Wire(repo, repo, everySystem(t)))

	added = slices.DeleteFunc(added, func(name string) bool { return slices.Contains(core, name) })

	// Compacted, the tracker counting once for the two seams it fills.
	slices.Sort(added)
	testenv.Equal(t, slices.Sorted(slices.Values(config.SystemNames())), slices.Compact(added),
		"a system the wiring has is one no settings file spells")
}

// No two resolvers mark their rows alike, and each mark is the one column the
// picker pads for. Structural for the same reason: a command draws one row.
func TestTheResolversMarksAreDistinctAndOneColumnWide(t *testing.T) {
	repo := t.TempDir()

	marks := map[string]string{}
	for _, r := range Wire(repo, repo, everySystem(t)).Resolvers {
		icon := r.Icon()
		by, taken := marks[icon]
		require.Falsef(t, taken, "%s and %s both mark their rows %q", by, r.Name(), icon)
		require.Equalf(t, 1, utf8.RuneCountInString(icon),
			"%s marks its rows with more than the one column the picker pads for", r.Name())
		marks[icon] = r.Name()
	}
}

// everySystem is the settings of a machine that named every system, read the way
// work reads them. Nothing holds the name internal/config spells and the name
// the implementation goes by together, so this names both.
func everySystem(t *testing.T) config.Config {
	t.Helper()
	testenv.Settings(t, `systems = ["`+strings.Join(config.SystemNames(), `", "`)+"\"]\n")
	return load(t)
}

// load is the settings on this machine, read through Load rather than taken from
// [config.Default], so that what writing nothing gets is judged.
func load(t *testing.T) config.Config {
	t.Helper()
	cfg, err := config.Load()
	require.NoError(t, err, "the settings")
	return cfg
}

// wired is every system a wiring holds, under the names they go by.
func wired(systems work.Systems) []string {
	return slices.Concat(names(systems.Resolvers), names(systems.Actions), names(systems.Openers))
}

// names are the systems under the names they go by.
func names[T worktree.System](systems []T) []string {
	var under []string
	for _, s := range systems {
		under = append(under, s.Name())
	}
	return under
}
