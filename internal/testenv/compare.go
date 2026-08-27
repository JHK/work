package testenv

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// Equal fails the test unless got compares the same as want, saying what the
// case promises and then go-cmp's diff of the two. Nil and empty count as the
// same. Options are go-cmp's, for a value that will not compare without one:
// [cmpopts.IgnoreUnexported] and [cmpopts.EquateErrors] are the ones this
// module's values ask for.
func Equal[T any](t *testing.T, want, got T, says string, opts ...cmp.Option) {
	t.Helper()
	if d := cmp.Diff(want, got, append([]cmp.Option{cmpopts.EquateEmpty()}, opts...)...); d != "" {
		t.Errorf("%s (-want +got):\n%s", says, d)
	}
}
