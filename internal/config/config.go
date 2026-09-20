// Package config reads the settings behind the choices work makes on a user's
// behalf: the user's file, over the compiled-in defaults. It also names that
// file, whether or not it is there.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/JHK/work-cli/internal/worktree"
)

// Config is every setting work reads: the integrations switched on, then one
// field per table.
type Config struct {
	Integrations []worktree.IntegrationName
	Worktree     Worktree
	Beads        Beads
	Claude       Claude
}

const (
	defaultTicket = "{{.ID}}{{with .Slug}}-{{.}}{{end}}"

	dirKey         = "worktree.directory"
	beadsBranchKey = "beads.branch"
)

var defaultBeads = Beads{BranchPattern: must("pattern", defaultTicket, parsePattern, (*Pattern).compileMatcher)}

// Default is what an unset key falls back to. The integrations list is left
// empty, which is every integration off.
func Default() Config {
	return Config{
		Worktree: defaultWorktree,
		Beads:    defaultBeads,
		Claude:   defaultClaude,
	}
}

// Load reads the file over the defaults, key by key. A file that is not there is
// no error; one that cannot be read, names a key work does not know, or carries
// an unusable value, is.
func Load() (Config, error) {
	c := Default()
	path := UserFile()
	if path != "" {
		if err := decode(path, &c); err != nil {
			return Config{}, err
		}
	}
	if key, err := c.validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %s: %w", path, key, err)
	}
	return c, nil
}

// UserFile is the settings file, named whether or not it is there. A machine
// with nowhere to keep it has none, and answers with the empty path.
func UserFile() string {
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

// decode reads the file over what the defaults left, leaving every key the file
// does not name alone.
func decode(path string, c *Config) error {
	md, err := toml.DecodeFile(path, c)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	// A key nothing decoded is a typo of one that would have, not a value to drop.
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		return fmt.Errorf("%s: unknown setting %s", path, undecoded[0])
	}
	// toml matches a key to a field case-insensitively, so two spellings of one
	// key would race to set it. Only the documented spelling is that key.
	for _, key := range md.Keys() {
		if name := key.String(); name != strings.ToLower(name) {
			return fmt.Errorf("%s: unknown setting %s", path, name)
		}
	}
	return nil
}

// validate names the key work cannot use the value of, and why. It also
// compiles the matcher of the tracker's pattern.
func (c *Config) validate() (string, error) {
	if err := c.validateIntegrations(); err != nil {
		return integrationsKey, err
	}
	c.Integrations = c.switchedOn()
	if err := c.Beads.BranchPattern.compileMatcher(); err != nil {
		return beadsBranchKey, err
	}
	if err := c.Claude.validate(); err != nil {
		return commandKey, err
	}
	if err := c.Worktree.validate(); err != nil {
		return dirKey, err
	}
	return "", nil
}
