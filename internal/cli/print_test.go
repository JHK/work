package cli

import (
	"errors"
	"testing"
)

// refusing is a writer that takes nothing, standing in for a closed pipe.
type refusing struct{}

func (refusing) Write([]byte) (int, error) { return 0, errors.New("refused") }

// What a verb did is said, or the invocation reports that it could not be.
func TestARefusedWriteFailsTheVerb(t *testing.T) {
	for _, args := range [][]string{{"move", "scratch", "settled"}, {"remove", "scratch"}} {
		t.Run(args[0], func(t *testing.T) {
			s := repository(t)
			s.opened("scratch")

			r := s.reads(refusing{}, args...)

			r.refused(t, "refused")
		})
	}
}
