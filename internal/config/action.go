package config

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// Action is what a worktree is handed over to where no flag names it: one key
// for a worktree just created, one for one already there. Each falls back to
// defaultAction on its own, so a configuration may set one and leave the other.
type Action struct {
	CreateName ActionName `toml:"create"`
	EnterName  ActionName `toml:"enter"`
}

const (
	createKey = "action.create"
	enterKey  = "action.enter"
)

// ActionName is one of the actions a worktree can be handed to, under the name
// the command line's own flag goes by.
type ActionName string

const (
	ActionClaude ActionName = "claude"
	ActionShell  ActionName = "shell"
)

// actionNames are the actions on here: what an [action] key may name.
func (c Config) actionNames() []string {
	if c.Claude.Enabled {
		return []string{string(ActionClaude), string(ActionShell)}
	}
	return []string{string(ActionShell)}
}

// renamed are the names an action used to go by, so a file written before a
// rename is told which name to write instead. One pair covers the action's key,
// its flag and its table.
var renamed = map[ActionName]ActionName{"agent": ActionClaude}

// Renamed are the names this action used to go by.
func Renamed(now ActionName) []ActionName {
	var was []ActionName
	for old, is := range renamed {
		if is == now {
			was = append(was, old)
		}
	}
	slices.Sort(was)
	return was
}

// Create is the action a worktree just created opens on. An unset key is the
// compiled-in one, so a Config that never reached Load still names an action.
func (a Action) Create() ActionName {
	return cmp.Or(a.CreateName, defaultAction.CreateName)
}

// Enter is the action a worktree already there opens on.
func (a Action) Enter() ActionName {
	return cmp.Or(a.EnterName, defaultAction.EnterName)
}

// validate names the key work cannot use the value of, and why.
func (a *Action) validate(names []string) (string, error) {
	if err := a.CreateName.validate(names); err != nil {
		return createKey, err
	}
	if err := a.EnterName.validate(names); err != nil {
		return enterKey, err
	}
	return "", nil
}

// validate refuses a name no action on here goes by. An empty value asks for
// the default back.
func (n ActionName) validate(names []string) error {
	if n == "" || slices.Contains(names, string(n)) {
		return nil
	}
	if now, ok := renamed[n]; ok {
		return fmt.Errorf("%q is now %q", n, now)
	}
	return fmt.Errorf("%q is not an action; the actions are %s", n, strings.Join(names, ", "))
}

// defaultAction is what work opens on where neither key names anything.
var defaultAction = Action{CreateName: ActionShell, EnterName: ActionShell}
