// Package config reads the settings behind the choices work makes on a user's
// behalf: the repository's file and the user's, merged over the compiled-in
// defaults.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config is every setting work reads, one field per table.
type Config struct {
	Worktree Worktree
	Branch   Branch
	Action   Action
	Agent    Agent
	Open     Open
}

type Worktree struct {
	Directory string
}

const (
	defaultDirectory   = ".worktrees"
	defaultTicket      = "{{.ID}}{{with .Slug}}-{{.}}{{end}}"
	defaultPullRequest = "pr-{{.Number}}"

	dirKey         = "worktree.directory"
	ticketKey      = "branch.ticket"
	pullRequestKey = "branch.pull-request"
)

// Dir is where a worktree is created, relative to the repository root. An unset
// Directory is the default, so a Config that never reached Load still names one.
func (w Worktree) Dir() string {
	if w.Directory == "" {
		return defaultDirectory
	}
	return w.Directory
}

// RepoFile is the repository's settings, at its root and checked in, so every
// clone gets them. The user's are userFile.
const RepoFile = ".work.toml"

// defaults are the compiled-in patterns, bound once.
var defaults = Branch{
	TicketPattern:      mustPattern(defaultTicket, ticketValues),
	PullRequestPattern: mustPattern(defaultPullRequest, pullRequestValues),
}

// Default is what an unset key falls back to.
func Default() Config {
	return Config{
		Worktree: Worktree{Directory: defaultDirectory},
		Branch:   defaults,
		Action:   defaultAction,
		Agent:    defaultAgent,
		Open:     defaultOpen,
	}
}

// mustPattern binds a compiled-in default, which cannot be at fault.
func mustPattern(text string, v values) Pattern {
	p, err := parsePattern(text)
	if err == nil {
		err = p.bind(v)
	}
	if err != nil {
		panic(fmt.Sprintf("config: default pattern %q %v", text, err))
	}
	return p
}

// Load merges the user's file and then the repository's over the defaults, key
// by key. A file that is not there is no error; one that cannot be read, names a
// key work does not know, or carries an unusable value, is.
func Load(repo string) (Config, error) {
	c := Default()
	// Which file last set each key, so that refusing a value names the file that
	// gave it rather than whichever layer was read last.
	from := map[string]string{}
	for _, path := range files(repo) {
		md, err := decode(path, &c)
		if err != nil {
			return Config{}, err
		}
		for _, key := range md.Keys() {
			from[key.String()] = path
		}
	}
	// Once merged, not per layer: a value the layer above replaces is not the one
	// work uses, and a directory is judged against the repository it is used in.
	if key, err := c.validate(repo); err != nil {
		return Config{}, fmt.Errorf("%s: %s: %w", from[key], key, err)
	}
	return c, nil
}

// files are the layers above the defaults, lowest first.
func files(repo string) []string {
	var paths []string
	if user := userFile(); user != "" {
		paths = append(paths, user)
	}
	return append(paths, filepath.Join(repo, RepoFile))
}

// userFile is settings that follow one user from repository to repository. A
// machine with nowhere to keep them has none.
func userFile() string {
	// Not os.UserConfigDir, which reads XDG_CONFIG_HOME on Unix alone.
	if dir := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(dir) {
		return filepath.Join(dir, "work", "config.toml")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "work", "config.toml")
}

// decode reads one layer over what the layers below it left, leaving every key
// the file does not name alone.
func decode(path string, c *Config) (toml.MetaData, error) {
	md, err := toml.DecodeFile(path, c)
	if errors.Is(err, fs.ErrNotExist) {
		return toml.MetaData{}, nil
	}
	if err != nil {
		return toml.MetaData{}, fmt.Errorf("%s: %w", path, err)
	}
	// A key nothing decoded is a typo of one that would have, not a value to drop.
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		return md, fmt.Errorf("%s: unknown setting %s", path, undecoded[0])
	}
	// toml matches a key to a field case-insensitively, so two spellings of one
	// key would race to set it. Only the documented spelling is that key.
	for _, key := range md.Keys() {
		if name := key.String(); name != strings.ToLower(name) {
			return md, fmt.Errorf("%s: unknown setting %s", path, name)
		}
	}
	return md, nil
}

// validate names the key work cannot use the value of, and why. It also ties
// each pattern and command to the values its key has, which is where either is
// judged.
func (c *Config) validate(repo string) (string, error) {
	if err := c.Branch.TicketPattern.bind(ticketValues); err != nil {
		return ticketKey, err
	}
	if err := c.Branch.PullRequestPattern.bind(pullRequestValues); err != nil {
		return pullRequestKey, err
	}

	if key, err := c.Action.validate(); err != nil {
		return key, err
	}
	if key, err := c.Agent.validate(); err != nil {
		return key, err
	}
	if key, err := c.Open.validate(); err != nil {
		return key, err
	}

	dir := c.Worktree.Directory
	// A worktree needs a directory of its own inside the repository, so that one
	// entry can tell git to ignore every worktree at once.
	if !filepath.IsLocal(dir) || filepath.Clean(dir) == "." {
		return dirKey, fmt.Errorf("%q is not a directory inside the repository", dir)
	}
	if top, _, _ := strings.Cut(filepath.ToSlash(filepath.Clean(dir)), "/"); top == ".git" {
		return dirKey, fmt.Errorf("%q is inside git's own directory", dir)
	}
	// IsLocal is lexical, and .work.toml arrives from whoever is cloned.
	if !contains(repo, filepath.Join(repo, dir)) {
		return dirKey, fmt.Errorf("%q resolves outside the repository", dir)
	}
	return "", nil
}

// contains reports whether path sits below root once symlinks are resolved. What
// is not on disk yet cannot lead anywhere, so only what resolves is judged.
func contains(root, path string) bool {
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return true
	}
	if root, err = filepath.EvalSymlinks(root); err != nil {
		return true
	}
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != "." && filepath.IsLocal(rel)
}
