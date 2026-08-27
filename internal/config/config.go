// Package config reads the settings behind the choices work makes on a user's
// behalf: the user's file, over the compiled-in defaults. It also hands that
// file to an editor, bringing it into being where it is not there yet.
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

// Config is every setting work reads: the systems switched on, then one field
// per table.
type Config struct {
	Systems  []string
	Worktree Worktree
	Branch   Branch
	Claude   Claude
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

// defaults are the compiled-in patterns, bound once.
var defaults = Branch{
	TicketPattern:      mustPattern(defaultTicket, ticketValues),
	PullRequestPattern: mustPattern(defaultPullRequest, pullRequestValues),
}

// Default is what an unset key falls back to. The systems list is left empty,
// which is every system off.
func Default() Config {
	return Config{
		Worktree: Worktree{Directory: defaultDirectory},
		Branch:   defaults,
		Claude:   defaultClaude,
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

// Load reads the file over the defaults, key by key. A file that is not there is
// no error; one that cannot be read, names a key work does not know, or carries
// an unusable value, is.
func Load() (Config, error) {
	c, _, err := read()
	return c, err
}

// read is Load, keeping which keys the file set, which a dump names and a
// refusal points at.
func read() (Config, map[string]string, error) {
	c := Default()
	from := map[string]string{}
	if path := userFile(); path != "" {
		md, err := decode(path, &c)
		if err != nil {
			return Config{}, nil, err
		}
		for _, key := range md.Keys() {
			from[key.String()] = path
		}
	}
	if key, err := c.validate(); err != nil {
		return Config{}, nil, fmt.Errorf("%s: %s: %w", from[key], key, err)
	}
	return c, from, nil
}

// userFile is the settings file, named whether or not it is there. A machine
// with nowhere to keep it has none, and answers with the empty path.
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

// renamed are the names a table used to go by, so a file written before a rename
// is told which name to write instead.
var renamed = map[string]string{"agent": ClaudeSystem}

// decode reads the file over what the defaults left, leaving every key the file
// does not name alone.
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
		key := undecoded[0]
		if now, ok := renamed[key[0]]; ok {
			return md, fmt.Errorf("%s: the [%s] table is now [%s]", path, key[0], now)
		}
		return md, fmt.Errorf("%s: unknown setting %s", path, key)
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
// each pattern and command to the values its key has.
func (c *Config) validate() (string, error) {
	if err := c.validateSystems(); err != nil {
		return systemsKey, err
	}
	c.Systems = c.resolved()
	if err := c.Branch.TicketPattern.bind(ticketValues); err != nil {
		return ticketKey, err
	}
	if err := c.Branch.PullRequestPattern.bind(pullRequestValues); err != nil {
		return pullRequestKey, err
	}

	if key, err := c.Claude.validate(); err != nil {
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
	return "", nil
}
