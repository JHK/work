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
	ActionEditor ActionName = "editor"
	ActionDiff   ActionName = "diff"
	// ActionAsk is the choice between the four above rather than a fifth command:
	// it draws the screen and opens on what came back.
	ActionAsk ActionName = "ask"
)

// actionNames are the names an [action] key may hold, which is also how they
// read in a refusal.
var actionNames = []string{
	string(ActionClaude), string(ActionShell), string(ActionEditor), string(ActionDiff), string(ActionAsk),
}

// renamed are the names an action used to go by, so a file written before a
// rename is told which name to write instead of being told what it names is
// unknown. An implementation's table of keys is named for the action, so one
// pair covers a file spelling either the old way.
var renamed = map[ActionName]ActionName{"agent": ActionClaude}

// Renamed are the names this action used to go by. A key and a flag spell the
// one name, so a flag it used to answer to is this same table read backwards.
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
	if n == "" || slices.Contains(actionNames, string(n)) {
		return nil
	}
	if now, ok := renamed[n]; ok {
		return fmt.Errorf("%q is now %q", n, now)
	}
	return fmt.Errorf("%q is not an action; the actions are %s", n, strings.Join(actionNames, ", "))
}

// defaultAction is what work opens on where neither key names anything: the
// shell either way, navigation being the remit. Neither name may be one a system
// has to be on for, these being what a repository that configured nothing gets.
// A repository that turns the agent on names claude in create.
var defaultAction = Action{CreateName: ActionShell, EnterName: ActionShell}
