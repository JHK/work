// Package forge is the adapter over the gh CLI, work's one question to the
// host: which pull requests are open.
package forge

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/JHK/work-cli/internal/run"
)

// PR is a pull request as far as work is concerned: the number that names its
// worktree, and the title a row shows.
type PR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
}

// prLimit caps the list; gh stops at 30 by default and has no unlimited form.
const prLimit = "200"

// Open lists the open pull requests, drafts included, of the repository at
// remote. The remote is named rather than left to gh, which resolves a checkout
// with several to whichever it prefers, and would list pull requests whose head
// the fetch could then not find. An absent, unauthenticated or offline gh is an
// error like any other; callers decide whether they can go on without the list.
func Open(repo, remote string) ([]PR, error) {
	if remote == "" {
		return nil, errors.New("gh: no remote to list pull requests of")
	}
	out, err := run.Output(repo, "gh", "pr", "list", "--repo", remote,
		"--state", "open", "--limit", prLimit, "--json", "number,title")
	if err != nil {
		return nil, err
	}
	var prs []PR
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return nil, fmt.Errorf("gh: %w", err)
	}
	return prs, nil
}
