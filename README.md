# work

<!-- This file is the brief, not a description of the binary. It states the tool
     `work` is meant to be, and the build is measured against it. Extend it when
     the intent changes and close the delta in code; never edit it down to match
     what ships. What ships is docs/references/. -->

`work` is a smarter `cd` for git worktrees. It knows which worktrees a repository has open and which tickets and pull requests are waiting, so the place you meant to work is a few keystrokes away.

Navigation is the whole remit. `work` provisions the worktree and hands the terminal to whatever it opens on: your shell, or a command. The second form makes it a launcher for [agentic work](docs/explanation/worktree-per-ticket.md), one keystroke from a ticket to an agent running in its own checkout.

## Install

Build it from source. There is no package yet.

```
mise run install
```

`git` is the only dependency. Other [tooling](docs/references/tools.md) is reached for as needed. Every build and install task: [mise tasks](docs/references/mise-tasks.md).

## Use

```
work
```

Choose from the repository's worktrees, its ready tickets and its open pull requests.

- **With a worktree:** a shell in it, with the lines that resume the sessions it carries.
- **Without one:** the worktree is created, the ticket claimed, and the configured launcher invoked in it.

```
work <name>
```

A worktree name, a ticket id or a pull request skips the chooser. See also [the command line](docs/references/cli.md).

## Still in development

The tool above is the target. What ships is locked to one instance of each integration, and [a missing one](docs/references/tools.md) fails its path rather than degrading it.

- **The launcher is `claude`.** There is nothing to configure: the command, its flags, the skills a fresh worktree opens on and the place session history is read from are all fixed. A new worktree that cannot start it is left created and claimed with nothing to hand over to.
- **The tracker is `bd`.** It names, vets and claims, and it creates the worktree too, so a name that is not a pull request resolves nowhere without it.
- **A pull request is a GitHub pull request on `origin`.** The branch is fetched from that one remote in that one forge's ref layout.
- **The shell is fish.** It is what `work` is built and used against, and the resume lines it prints are joined for pasting there and quoted for nothing else.
- **The picker is `fzf`.** The no-argument form needs it.
