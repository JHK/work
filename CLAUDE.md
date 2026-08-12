# CLAUDE.md

Agent instructions for `work-cli`: a Go CLI that turns a ticket, a pull request, or an open worktree into a git worktree to work in.

[README.md](README.md) is the brief: it states the tool `work` is meant to be, not what the binary does. Extend it when the intent changes; never edit it down to match the code. What ships is [docs/references/](docs/references/AGENTS.md).

## Before acting

Documentation under [docs/](docs/) is canonical: consult it before running commands, editing files, or answering questions about how things work.

Routing by question type:

| Question | Where |
|---|---|
| "What is the current state of X?" | [docs/references/](docs/references/AGENTS.md) |
| "What must not break when I change X?" | [docs/rules/](docs/rules/AGENTS.md) |
| "What has bitten past sessions in X?" | [docs/gotchas/](docs/gotchas/AGENTS.md) |
| "How do I do X?" | [docs/howto/](docs/howto/AGENTS.md) |
| "What are we agreeing to build for X?" | [docs/projects/](docs/projects/AGENTS.md) |
| "What is the plan for X?" | `bd show <epic>` |
| "Why is X shaped this way?" | [docs/explanation/](docs/explanation/AGENTS.md) |

Open the subfolder's `AGENTS.md` first, then skim the folder's filenames and title lines and open only what the question needs. Never pre-load a folder.

## Comments

Inside a function body, comment only where the intent is not evident from the code, in as few words as that takes, never past two lines.

A doc comment says what the symbol is or does for whoever calls it, and stops there. Only one on an exported symbol may run longer, and only where a caller needs that behaviour.

No comment justifies a design choice, restates the code beneath it, or explains a package's architecture. That reasoning goes in [docs/explanation/](docs/explanation/AGENTS.md), its invariants in [docs/rules/](docs/rules/AGENTS.md), and is dropped rather than moved where those already carry it.
