package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/JHK/work-cli/internal/worktree"
)

// Edit hands the settings file to an editor, $VISUAL else $EDITOR, bringing the
// file and the directory it sits in into being. Everything that can refuse does
// so first.
func Edit() (worktree.Handoff, error) {
	editor := strings.Fields(os.Getenv("VISUAL"))
	if len(editor) == 0 {
		editor = strings.Fields(os.Getenv("EDITOR"))
	}
	if len(editor) == 0 {
		return worktree.Handoff{}, errors.New("neither $VISUAL nor $EDITOR names an editor to open your settings in")
	}
	path := userFile()
	if path == "" {
		return worktree.Handoff{}, errors.New("this machine names neither $XDG_CONFIG_HOME nor a home directory, so there is nowhere to keep your settings")
	}
	if err := create(path); err != nil {
		return worktree.Handoff{}, err
	}
	return worktree.Handoff{Dir: filepath.Dir(path), Run: append(editor, path)}, nil
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
