// Package settings answers for work config: the settings work resolved in a
// repository, and the user's own file. A front end asks for the dump or the
// handoff and reaches neither git nor the file itself.
package settings

import (
	"cmp"
	"errors"
	"os"
	"path/filepath"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/worktree"
)

// Dump is the configuration work reads in the repository dir sits in, rendered
// as TOML with the layer behind each key.
func Dump(dir string) (string, error) {
	repo, err := git.Root(dir)
	if err != nil {
		return "", err
	}
	return config.Dump(repo)
}

// Edit hands the user's settings file to an editor, bringing the file and the
// directory it sits in into being where neither is there yet. The editor is
// $VISUAL else $EDITOR, never a setting: this is the file such a setting would
// be written in. Everything that can refuse does so before anything is created.
func Edit() (worktree.Handoff, error) {
	editor := cmp.Or(os.Getenv("VISUAL"), os.Getenv("EDITOR"))
	if editor == "" {
		return worktree.Handoff{}, errors.New("neither $VISUAL nor $EDITOR names an editor to open your settings in")
	}
	path := config.UserFile()
	if path == "" {
		return worktree.Handoff{}, errors.New("this machine names neither $XDG_CONFIG_HOME nor a home directory, so there is nowhere to keep your settings")
	}
	if err := create(path); err != nil {
		return worktree.Handoff{}, err
	}
	return worktree.Handoff{Dir: filepath.Dir(path), Run: []string{editor, path}}, nil
}

// create brings an empty settings file into being, leaving one already there as
// it is.
func create(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}
