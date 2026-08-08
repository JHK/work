package config

import (
	"cmp"
	"fmt"
	"slices"
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
	ActionAgent  ActionName = "agent"
	ActionShell  ActionName = "shell"
	ActionEditor ActionName = "editor"
	ActionDiff   ActionName = "diff"
)

// actionNames are the names an [action] key may hold, and actionList is how
// they read in a refusal.
var actionNames = []ActionName{ActionAgent, ActionShell, ActionEditor, ActionDiff}

const actionList = "agent, shell, editor, diff"

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
func (a *Action) validate() (string, error) {
	if err := a.CreateName.validate(); err != nil {
		return createKey, err
	}
	if err := a.EnterName.validate(); err != nil {
		return enterKey, err
	}
	return "", nil
}

// validate refuses a name no action goes by. Load starts from Default, so an
// empty value is a file asking for the default back rather than an unset key.
func (n ActionName) validate() error {
	if n == "" || slices.Contains(actionNames, n) {
		return nil
	}
	return fmt.Errorf("%q is not an action; the actions are %s", n, actionList)
}

// defaultAction is what work opened on before either was a setting: a worktree
// just created is handed to the agent, one already there to the shell.
var defaultAction = Action{CreateName: ActionAgent, EnterName: ActionShell}
