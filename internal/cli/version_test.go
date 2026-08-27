package cli

import (
	"testing"
)

func TestTheVersionAsksNoRepository(t *testing.T) {
	s := repository(t)
	s.Dir = t.TempDir()

	r := s.run("--version")

	r.came(t, result{Out: versionLine})
}
