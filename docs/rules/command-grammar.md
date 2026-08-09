# Command grammar

The invariants of the first argument position and of where a flag is declared. What each verb does is [the command line](../references/cli.md).

## R1 — A command wins the first position

A worktree whose name is a command's cannot be reached by that name; `work switch <name>` reaches it. Every other first word, a name, a flag the root does not declare, or nothing at all, reaches `switch`.

*Enforced by:* `dispatch`, `rootsOwn` and `rootFlag` in `internal/cli/dispatch.go`, which rewrite the argv before cobra executes the tree, and `internal/cli/dispatch_test.go`.

## R2 — Flags are declared on their verb

`--help` and `--version`, shorthands included, are the root's; no other flag is. The verb that uses a flag declares it, and a verb that does not declare one refuses it.

*Enforced by:* `openOn` in `internal/cli/cli.go`, which declares the shared set once per verb that opens something, and `TestOpenOnFlagsAreSwitchs` and `TestCommandRejects` in `internal/cli/cli_test.go`.
