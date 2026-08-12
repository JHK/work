// Package settings answers for work config: the settings work resolved in a
// repository, and the user's own file. It is the one verb whose subject is a
// file rather than a worktree, so it sits beside the core rather than in it, and
// it is where knowing git and knowing where the user's file lives stays: a front
// end asks for the dump or the handoff and reaches neither.
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
// as TOML with the layer behind each key. A repository is what the layers are
// read against, so one that will not answer is refused here as it is at every
// other verb.
func Dump(dir string) (string, error) {
	repo, err := git.Root(dir)
	if err != nil {
		return "", err
	}
	return config.Dump(repo)
}

// Edit hands the user's settings file to an editor, bringing the file and the
// directory it sits in into being where neither is there yet, so an editor that
// creates neither still opens on something.
//
// The editor is $VISUAL else $EDITOR, read here rather than named by a setting:
// the file this opens is where such a setting would have been written, so
// needing it set to fix it is a door that locks from the inside.
//
// Everything that can refuse does so before anything is created.
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
	// The directory was just made, so there is one to hand over in whether or not
	// the editor cares which it is started from.
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
