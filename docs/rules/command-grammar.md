# Command grammar

The invariants of the first argument position and of where a flag is declared. What each verb does is [the command line](../references/cli.md).

## R1 — A command wins the first position

A worktree whose name is a command's cannot be reached by that name; `work go <name>` reaches it. Every other first word, a name, a flag the root does not declare, or nothing at all, reaches `go`. A flag the root hands down to every verb holds no position, and neither does the word it takes its value from: the first is the one behind both.

*Enforced by:* `dispatch`, `rootsOwn` and `declared` in `internal/cli/dispatch.go`, which rewrite the argv before cobra executes the tree, and `internal/cli/dispatch_test.go`.

## R2 — Flags are declared on their verb

`--help`, `--log-level` and `--version`, shorthands included, are the root's; no other flag is. `--log-level` is the one the root hands down, so every verb carries it. Neither it nor `--version` takes a shorthand: one letter that could name either the version or the level names neither. The verb that uses any other flag declares it, and a verb that does not declare one refuses it.

*Enforced by:* `openOn` in `internal/cli/cli.go` and `logging` in `internal/cli/loglevel.go`, which declare the shared set once per verb that opens something and the handed-down flag once for the tree, and `TestWhereEachSystemsFlagIsDeclared` and `TestCommandRejects` in `internal/cli/flags_test.go`, `TestNeitherTheVersionNorTheLevelTakesAShorthand` in `internal/cli/loglevel_test.go`.
