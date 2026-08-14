# work

<!-- The brief, not a description of the binary. Never edit it down to match what
     ships; see CLAUDE.md. -->

`work` is a smarter `cd` for git worktrees. It knows which worktrees a repository has open and which tickets and pull requests are waiting, so the place you meant to work is a few keystrokes away.

Navigation is the whole remit. `work` provisions the worktree and hands it to whatever it opens on: your shell, or a command. The second form makes it a launcher for [agentic work](docs/explanation/work-loop.md), one keystroke from a ticket to an agent running in its own checkout.

## Install

Build it from source. There is no package yet.

```
mise run install
```

Then one line in `config.fish`:

```
work init fish | source
```

`git` is the only dependency. Other [tooling](docs/references/systems.md) is reached for as needed. Every build and install task: [mise tasks](docs/references/mise-tasks.md).

## Use

```
work
```

Choose from the repository's worktrees, its ready tickets and its open pull requests.

- **With a worktree:** what you set it to open on, a shell by default.
- **Without one:** the worktree is created first, then opens the same way.

```
work <name>
```

A worktree name, a ticket id or a pull request skips the chooser. See also [the command line](docs/references/cli.md).

## Still in development

What ships is locked to one instance of each integration, and [a missing one](docs/references/systems.md) fails its path rather than degrading it.

- **The resolvers are `beads` and `github`.** A ticket is a bead over `bd`; a pull request one of `origin`'s, fetched in that one forge's ref layout.
- **The agent opener is `claude`.**
- **The shell integration is fish.**
- **The picker is `fzf`.** The no-argument form needs it.

## License

MIT. See [LICENSE](LICENSE).
