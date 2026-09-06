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

// An integration wired off a name [config.IntegrationNames] leaves out is one
// no settings file reaches. Structural like R3 and R4: no command asserts an
// "every".
func TestTheSettingsSpellEveryIntegrationTheWiringHas(t *testing.T) {
	repo := worktree.Repo(t.TempDir())
	from := worktree.Path(repo)
	// Read before the settings are written: each takes a settings home of its own.
	core := wired(Wire(repo, from, load(t)))
	added := wired(Wire(repo, from, everyIntegration(t)))

	added = slices.DeleteFunc(added, func(name worktree.IntegrationName) bool { return slices.Contains(core, name) })

	// Compacted, the tracker counting once for the two seams it fills.
	slices.Sort(added)
	spelled := config.IntegrationNames()
	want := make([]worktree.IntegrationName, 0, len(spelled))
	for _, name := range spelled {
		want = append(want, worktree.IntegrationName(name))
	}
	slices.Sort(want)
	testenv.Equal(t, want, slices.Compact(added),
		"an integration the wiring has is one no settings file spells")
}

// Structural like the case above: a command draws one row, never the set.
func TestTheResolversMarksAreDistinctAndOneColumnWide(t *testing.T) {
	repo := worktree.Repo(t.TempDir())

	// The core's own answer marks rows too, and no settings file names it.
	chain := work.Env{Seams: Wire(repo, worktree.Path(repo), everyIntegration(t))}.Chain()

	marks := map[string]worktree.IntegrationName{}
	for _, r := range chain {
		icon := r.Icon()
		by, taken := marks[icon]
		require.Falsef(t, taken, "%s and %s both mark their rows %q", by, r.Name(), icon)
		require.Equalf(t, 1, utf8.RuneCountInString(icon),
			"%s marks its rows with more than the one column the picker pads for", r.Name())
		marks[icon] = r.Name()
	}
}

// everyIntegration is the settings of a machine that named every integration,
// read the way work reads them. Nothing holds the name internal/config spells
// and the name the implementation goes by together, so this names both.
func everyIntegration(t *testing.T) config.Config {
	t.Helper()
	testenv.Settings(t, `integrations = ["`+strings.Join(config.IntegrationNames(), `", "`)+"\"]\n")
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

// wired is every integration a wiring holds, under the names they go by.
func wired(integrations work.Seams) []worktree.IntegrationName {
	return append(slices.Concat(names(integrations.Resolvers), names(integrations.Actions)),
		integrations.Handback.Name())
}

func names[T worktree.Named](integrations []T) []worktree.IntegrationName {
	var under []worktree.IntegrationName
	for _, s := range integrations {
		under = append(under, s.Name())
	}
	return under
}
