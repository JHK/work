# Verb-first command grammar

*Epic: work-cli-1g1*

`work` says what it does before it says what to. Every capability is a verb in git's vocabulary carrying only the flags it has use for, and the bare `work <name>` stays a shortcut for the verb reached for constantly.

Everything `work` does is one command and a flag today, and the flag set is where the strain shows. `--force` is meaningless on every path but `--delete`, yet sits in the same `--help`; mutual exclusion is declared by hand between an ever-growing set; `--help` reads as a list of modes rather than a set of things you can do. The identifier has the same problem from the other side. `init` is a subcommand, so no worktree named `init` can be reached, and every capability added makes that worse for as long as the name and the verb share one position.

## The verbs

| Invocation | Does |
|---|---|
| `work switch <name>` | enters the worktree of a worktree name, a ticket or a pull request, creating it if there is none |
| `work add <name>` | creates a worktree on a new branch of that name, verbatim, asking no tracker |
| `work remove <name>` | removes a worktree and the branch it had checked out |
| `work init fish` | prints the shell integration |
| `work <name>` | `work switch <name>` |

`add` and `remove` are `git worktree`'s own verbs and refuse what it refuses: a name a branch already holds, an unclean worktree, a branch not merged. git has no verb for entering a worktree that exists, which is `work`'s whole remit, so `switch` is borrowed from the branch vocabulary.

Each verb carries its own flags and nothing else's. The open-on flags, `--agent`, `--shell`, `--editor`, `--diff` and `--ask`, belong to the two verbs that open something; `--no-claim` to `switch` alone, `add` having no ticket to decline; `--force` to `remove`. Only `--version` and `--help` are the root's.

## The bare form

`work` with no verb is `work switch`, argument or none. The shortcut lives in that position and nowhere else, which settles the collisions on its own: a verb always wins there, so `work add` is the creation verb and `work switch add` reaches a worktree named `add`. Every name is reachable through its verb, whatever verbs exist.

The no-argument forms are unchanged. `work` and `work switch` open the same picker, over the repository's worktrees, its open pull requests and its ready tickets; `work remove` opens it over the worktrees alone; `work add` needs a name.

## Completion

A tab press completes against the verb it follows, with no flag to read: `switch` offers what the picker offers, `remove` the worktrees alone, `add` nothing, the name being new. The bare position offers the verbs alone, so reaching a worktree by tab is `work switch <TAB>`.

## The reference documentation

[The command line](../references/cli.md) is a page of conditionals: which flags exclude which, what one flag means in the presence of another, what the picker offers under `--delete`. A verb owns its arguments, its flags, its refusals and its completion, so the page becomes a section per verb that reads straight through. Whether that branching is gone is how this work is judged.

## Out of scope

- Backwards compatibility. `--create`, `--delete` and a root-level `--force` are gone, and are not kept as aliases.
- A `list` verb, and what it would print.
- Short aliases for the verbs.
- What the picker offers, and the no-argument form's behaviour.
- Configuration keys and their names.
- The `w` abbreviation and anything else the shell integration installs.
