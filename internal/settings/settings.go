// Package settings answers for work config: the settings work resolved in a
// repository, and the user's own file. It is the one verb whose subject is a
// file rather than a worktree, which is why it sits beside the core rather than
// in it: the file stands where a worktree stands, and the action that opens it
// is reached through the same seam every worktree is handed over by.
package settings

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/git"
	"github.com/JHK/work-cli/internal/work"
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

// Edit hands the user's settings file to the editor action, bringing the file
// and the directory it sits in into being where neither is there yet, so an
// editor that creates neither still opens on something.
//
// Everything that can refuse does so before anything is created: the repository
// the settings are read against, the settings themselves, and the command.
func Edit(dir string, wire work.Wiring) (worktree.Handoff, error) {
	e, err := work.Open(dir, wire)
	if err != nil {
		return worktree.Handoff{}, err
	}
	path := config.UserFile()
	if path == "" {
		return worktree.Handoff{}, errors.New("this machine names neither $XDG_CONFIG_HOME nor a home directory, so there is nowhere to keep your settings")
	}

	// The file itself is what the editor is opened on, so it stands where a worktree
	// stands at every other verb, described by the same sources.
	h, err := e.Handoff(worktree.Tree{
		Place: worktree.Place{Name: filepath.Base(path)},
		Path:  path,
	}, string(config.ActionEditor))
	if err != nil {
		return worktree.Handoff{}, err
	}
	if err := create(path); err != nil {
		return worktree.Handoff{}, err
	}
	// An action changes into the worktree it was handed, and this one was handed a
	// file; the directory it sits in was just made, so there is one either way.
	h.Dir = filepath.Dir(path)
	return h, nil
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
