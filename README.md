# work

<!-- The brief, not a description of the binary. Never edit it down to match what
     ships; see CLAUDE.md. -->

`work` is a smarter `cd` for git worktrees, where the branch is the address. The place you meant to work is just a few keystrokes away.

Navigation is the whole remit. `work` provisions the worktree and hands it to whatever it opens on: your shell, or a command. The second form makes it a launcher for [agentic work](docs/explanation/work-loop.md), one keystroke from a ticket to an agent running in its own checkout.

## Install

Build it from source. There is no package yet.

```
mise run install
```

Then one line in your shell's startup file:

```
source <(work init bash)    # .bashrc
work init fish | source     # config.fish
source <(work init zsh)     # .zshrc, below compinit
```

`git` is the only dependency, and `fzf` is recommended for the chooser. Other [tooling](docs/references/systems.md) is reached for as needed. Every build and install task: [mise tasks](docs/references/mise-tasks.md).

## Use

```
work
```

Choose from the repository's worktrees. The systems you enable add to the list: ready tickets, open pull requests.

Every verb and argument: [the command line](docs/references/cli.md).

## Systems

Everything `work` reaches for beyond git is a system. Name the ones you work with in `~/.config/work/config.toml`:

```toml
systems = ["beads", "claude"]
```

A resolver answers before a worktree exists, turning an identifier into the place to work.

- [`beads`](docs/references/systems.md#beads): a ready ticket over `bd`
- [`github`](docs/references/systems.md#github): one open pull request

An action runs on the worktree that now exists.

- [`mise`](docs/references/systems.md#mise): trusts the new worktree
- [`claude`](docs/references/systems.md#claude): hands the worktree to the agent

Every setting: [configuration](docs/references/configuration.md).

## License

MIT. See [LICENSE](LICENSE).
