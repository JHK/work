# work

<!-- The brief, not a description of the binary. Never edit it down to match what
     ships; see CLAUDE.md. -->

`work` is a smarter `cd` for git worktrees. It knows which worktrees a repository has open and which tickets and pull requests are waiting, so the place you meant to work is a few keystrokes away.

Navigation is the whole remit. `work` provisions the worktree and hands the terminal to whatever it opens on: your shell, or a command. The second form makes it a launcher for [agentic work](docs/explanation/work-loop.md), one keystroke from a ticket to an agent running in its own checkout.

## Install

Build it from source. There is no package yet.

```
mise run install
```

Then one line in `config.fish`:

```
work init fish | source
```

`git` is the only dependency. Other [tooling](docs/references/tools.md) is reached for as needed. Every build and install task: [mise tasks](docs/references/mise-tasks.md).

## Use

```
work
```

Choose from the repository's worktrees, its ready tickets and its open pull requests.

- **With a worktree:** what you set it to open on, a shell by default.
- **Without one:** the worktree is created, the ticket claimed, and the configured launcher invoked in it.

```
work <name>
```

A worktree name, a ticket id or a pull request skips the chooser. See also [the command line](docs/references/cli.md).

## Still in development

What ships is locked to one instance of each integration, and [a missing one](docs/references/tools.md) fails its path rather than degrading it.

- **The tracker is `bd`.** It names, vets and claims, and it creates the worktree too, so a ticket resolves nowhere without it.
- **A pull request is a GitHub pull request on `origin`.** The branch is fetched from that one remote in that one forge's ref layout.
- **The shell is fish.** `work init fish` is the only integration it prints.
- **The picker is `fzf`.** The no-argument form needs it.

## License

MIT. See [LICENSE](LICENSE).
