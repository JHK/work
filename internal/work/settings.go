package work

import (
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
