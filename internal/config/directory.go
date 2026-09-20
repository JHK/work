package config

import (
	"errors"
	"fmt"

	"github.com/JHK/work-cli/internal/worktree"
)

// Worktree is the worktree table: where a worktree is created.
type Worktree struct {
	Directory Directory
}

const defaultDirectory = "{{.Repo}}/.worktrees"

var defaultWorktree = Worktree{Directory: must("directory", defaultDirectory, parseDirectory, (*Directory).validate)}

// Dir is where this repository's worktrees go.
func (w Worktree) Dir(repo worktree.Repo) string {
	return w.directory().render(repoValues(string(repo)))
}

func (w Worktree) validate() error { return w.directory().validate() }

// An unset directory is the compiled-in one, so a Config that never reached Load
// still names one.
func (w Worktree) directory() Directory { return w.Directory.or(defaultWorktree.Directory) }

// Directory is a [text/template] naming where a repository's worktrees go,
// rendered with that repository's path.
type Directory struct{ tmpl }

// repoValues is the data one render is given: the main checkout's path.
func repoValues(repo string) map[string]any { return map[string]any{"Repo": repo} }

// UnmarshalTOML reads one directory out of a settings file. [Directory.validate]
// judges the values later.
func (d *Directory) UnmarshalTOML(v any) error {
	text, ok := v.(string)
	if !ok {
		return errors.New("is not text")
	}
	parsed, err := parseDirectory(text)
	*d = parsed
	return err
}

func parseDirectory(text string) (Directory, error) {
	t, err := parseTmpl("directory", text, nil)
	return Directory{tmpl: t}, err
}

// or is the directory itself, or def where no file named one.
func (d Directory) or(def Directory) Directory {
	if d.t == nil {
		return def
	}
	return d
}

// validate renders the directory over the value it is given, so one that cannot
// render is refused at load.
func (d Directory) validate() error {
	if _, err := d.execute(repoValues("repo")); err != nil {
		return fmt.Errorf("%w; the value here is {{.Repo}}", err)
	}
	return nil
}
