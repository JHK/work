// Package github resolves the pull requests of the repository's origin into
// places to work. It is the only thing that speaks to the gh CLI.
package github

import (
	"cmp"
	"fmt"
	"regexp"
	"strconv"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/run"
	"github.com/JHK/work-cli/internal/worktree"
)

// Name is what this system goes by, and Command the CLI it goes through.
const (
	Name    = "github"
	Command = "gh"
)

// prURL matches an .../<owner>/<repo>/pull/<n> pull request URL.
var prURL = regexp.MustCompile(`^(?:[a-z]+://[^/]+/)?[^/\s]+/[^/\s]+/pull/([0-9]+)(?:[/?#].*)?$`)

// Resolver answers for the repository's pull requests.
type Resolver struct {
	repo    string
	pattern config.Branch
}

// New answers for the repository at repo, naming branches by the pull request
// pattern.
func New(repo string, branch config.Branch) Resolver {
	return Resolver{repo: repo, pattern: branch}
}

func (Resolver) Name() string { return Name }

// Icon marks a row that stands for a review.
func (Resolver) Icon() string { return "⇄" }

// Identify names the pull request behind what the core is holding: an identifier
// read by [Resolver.read], or a worktree, which is its own only where the branch
// is one the pattern names itself, so pr-007 does not stand for pr-7's worktree.
func (r Resolver) Identify(id string, o worktree.Open) (worktree.Place, error) {
	if o.None() {
		return r.read(id)
	}
	// An identifier to confirm where the core has one, else the branch itself.
	p, err := r.read(cmp.Or(id, o.Branch))
	if err != nil || p.Name != o.Branch {
		return worktree.Place{}, notOurs(o)
	}
	return p, nil
}

func notOurs(o worktree.Open) error {
	return fmt.Errorf("%w: no pull request is named by branch %q", worktree.ErrUnknown, o.Branch)
}

// read makes a place of a bare number, a pull request URL, or a name its own branch
// pattern could have produced. Anything else is another resolver's.
func (r Resolver) read(arg string) (worktree.Place, error) {
	n := ""
	if _, err := strconv.ParseUint(arg, 10, 32); err == nil {
		n = arg
	} else if m := prURL.FindStringSubmatch(arg); m != nil {
		n = m[1]
	} else if number, ok := r.pattern.NumberIn(arg); ok {
		// The worktree's own name, so that what the picker shows can be retyped.
		n = number
	}
	if n == "" {
		return worktree.Place{}, fmt.Errorf("%w: %q is no pull request of ours", worktree.ErrUnknown, arg)
	}
	// Canonicalise, so that 7 and 007 name one worktree and one refspec.
	i, err := strconv.ParseUint(n, 10, 32)
	if err != nil || i == 0 {
		return worktree.Place{}, fmt.Errorf("%q is not a pull request number", arg)
	}
	return r.place(strconv.FormatUint(i, 10), ""), nil
}

// Offer lists the open pull requests, drafts included. gh being absent,
// unauthenticated or offline costs these rows and nothing else.
func (r Resolver) Offer() ([]worktree.Place, error) {
	pulls, err := r.pulls()
	if err != nil {
		return nil, err
	}
	out := make([]worktree.Place, 0, len(pulls))
	for _, p := range pulls {
		out = append(out, r.place(strconv.Itoa(p.Number), p.Title))
	}
	return out, nil
}

// Prepare has nothing to ask: the branch is named from the number alone.
func (r Resolver) Prepare(p worktree.Place) (worktree.Place, error) { return p, nil }

// Create fetches the pull request's head and checks it out.
func (r Resolver) Create(p worktree.Place, path string) error {
	// A branch left behind by an earlier review is behind the PR head, so fetch
	// regardless and fall back to it only when the fetch cannot advance it.
	if err := git.Fetch(r.repo, fmt.Sprintf("pull/%s/head:%s", p.ID, p.Branch)); err != nil {
		if !git.HasBranch(r.repo, p.Branch) {
			return err
		}
	}
	return git.AddWorktree(r.repo, path, p.Branch)
}

// Supply spells out the pull request a worktree was made for.
func (r Resolver) Supply(t worktree.Tree) (worktree.Values, error) {
	subject := "PR #" + t.ID
	if t.Label != "" {
		subject += ": " + t.Label
	}
	return worktree.Values{worktree.SubjectValue: subject}, nil
}

// place names a pull request by the branch its worktree checks out.
func (r Resolver) place(number, title string) worktree.Place {
	branch := r.pattern.PullRequest(number)
	return worktree.Place{ID: number, Name: branch, Branch: branch, Label: title}
}

// pull is a pull request as far as work is concerned: the number that names its
// worktree, and the title a row shows.
type pull struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// prLimit caps the list; gh stops at 30 by default and has no unlimited form.
const prLimit = "200"

// pulls asks gh which pull requests are open, and asks nothing of a repository
// with no origin, which has none rather than an answer work could not get. The
// remote is named rather than left to gh, which resolves a checkout with several
// to whichever it prefers.
func (r Resolver) pulls() ([]pull, error) {
	remote := git.OriginURL(r.repo)
	if remote == "" {
		return nil, nil
	}
	return run.JSON[[]pull](r.repo, Command, "pr", "list", "--repo", remote,
		"--state", "open", "--limit", prLimit, "--json", "number,title")
}
