# CLAUDE.md

Agent instructions for `work-cli`: a Go CLI that turns a ticket, a pull request, or an open worktree into a git worktree to work in.

[README.md](README.md) is the brief: it states the tool `work` is meant to be, not what the binary does. Extend it when the intent changes; never edit it down to match the code. What ships is [docs/references/](docs/references/AGENTS.md).

## Before acting

Documentation under [docs/](docs/) is the canonical source for this repo. Consult it before running commands, editing files, or answering questions about how things work.

Routing by question type:

| Question | Subfolder |
|---|---|
| "What is the current state of X?" | [docs/references/](docs/references/AGENTS.md) |
| "What must not break when I change X?" | [docs/rules/](docs/rules/AGENTS.md) |
| "What has bitten past sessions in X?" | [docs/gotchas/](docs/gotchas/AGENTS.md) |
| "How do I do X?" | [docs/howto/](docs/howto/AGENTS.md) |
| "What is the plan for X?" | [docs/projects/](docs/projects/AGENTS.md) |
| "Why is X shaped this way?" | [docs/explanation/](docs/explanation/AGENTS.md) |

Each subfolder's `AGENTS.md` states what its type is for and how to judge relevance. Open that first, then the files it sends you to. Do not pre-load them all.
