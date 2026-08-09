package work

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/JHK/work-cli/internal/config"
	"github.com/JHK/work-cli/internal/git"
)

// Settings is the configuration work reads in the repository dir sits in,
// rendered as TOML with the layer behind each key. A repository is what the
// layers are read against, so one that will not answer is refused here as it is
// at every other verb.
func Settings(dir string) (string, error) {
	repo, err := git.Root(dir)
	if err != nil {
		return "", err
	}
	return config.Dump(repo)
}

// EditSettings hands the user's settings file to open.editor, bringing the file
// and the directory it sits in into being where neither is there yet, so an
// editor that creates neither still opens on something.
//
// Everything that can refuse does so before anything is created: the repository
// the settings are read against, the settings themselves, and the command.
func EditSettings(dir string) (Handoff, error) {
	e, err := Open(dir)
	if err != nil {
		return Handoff{}, err
	}
	path := config.UserFile()
	if path == "" {
		return Handoff{}, errors.New("this machine names neither $XDG_CONFIG_HOME nor a home directory, so there is nowhere to keep your settings")
	}
	run, err := e.editorOn(filepath.Base(path), path)
	if err != nil {
		return Handoff{}, err
	}
	if err := create(path); err != nil {
		return Handoff{}, err
	}
	// The directory the file sits in, as a worktree's handoff changes into the
	// worktree; it was just made, so there is one either way.
	return Handoff{Dir: filepath.Dir(path), Run: run}, nil
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
