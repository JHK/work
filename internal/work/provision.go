package work

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/JHK/work-cli/internal/beads"
	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/mise"
)

// Branch names the branch a target's worktree checks out. For a bead that is
// the id and a slug of its title, which is only knowable once bd has answered.
func (s State) Branch() (string, error) {
	if s.Target.Kind == KindPR {
		return s.Target.Name, nil
	}
	if s.TicketErr != nil {
		return "", s.TicketErr
	}
	if name := slug(s.Bead.Title); name != "" {
		return s.Target.ID + "-" + name, nil
	}
	return s.Target.ID, nil
}

// Provision makes the worktree exist and be usable. It is idempotent: an
// existing worktree is left alone.
func (e Env) Provision(s State) error {
	if s.Exists {
		return nil
	}
	branch, err := s.Branch()
	if err != nil {
		return err
	}

	switch s.Target.Kind {
	case KindPR:
		// A branch left behind by an earlier review is behind the PR head, so fetch
		// regardless and fall back to it only when the fetch cannot advance it.
		if err := git.Fetch(e.Repo, fmt.Sprintf("pull/%s/head:%s", s.Target.ID, branch)); err != nil {
			if !git.HasBranch(e.Repo, branch) {
				return err
			}
		}
		if err := git.AddWorktree(e.Repo, s.Target.Path, branch); err != nil {
			return err
		}
	default:
		if err := beads.CreateWorktree(e.Repo, s.Target.Path, branch); err != nil {
			return err
		}
	}

	mise.Trust(s.Target.Path)
	return nil
}

// Claim marks the bead as being worked.
func (e Env) Claim(t Target) error {
	if t.Kind != KindBead {
		return fmt.Errorf("%s is a pull request, not a bead", t.ID)
	}
	return beads.Claim(e.Repo, t.ID)
}

const slugLen = 40

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slug(title string) string {
	s := nonSlug.ReplaceAllString(strings.ToLower(title), "-")
	s = strings.Trim(s, "-")
	if len(s) > slugLen {
		s = s[:slugLen]
	}
	return strings.Trim(s, "-")
}
