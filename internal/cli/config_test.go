package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The verb with no sub-verb says what it carries instead of dumping, a dump
// being one thing config could go on to do rather than the thing it is.
func TestConfigWithoutASubVerb(t *testing.T) {
	s := repository(t)

	r := s.run("config")

	r.came(t, result{}, besides("Out"))
	for _, verb := range []string{"dump", "edit"} {
		require.Contains(t, r.Out, verb, "work config did not name the sub-verbs")
	}
}
