# Claude: Chain multiple instructions

Run `/simplify` after `/start` in the session a ticket's new worktree opens on.

## Prerequisites

- A ticket title carrying no `$` or `"`: `{{.Subject}}` is interpolated into a shell string.
- A decision on failure: `&&` skips the second instruction when the first exits non-zero, `;` in its place runs it either way.
- A pull request's worktree handed back: the one key carries every worktree, and every element below is guarded on `beads`.

## Steps

1. Set [`claude.command`](../references/configuration.md#keys) to the chain, wrapped in `sh -c`:

   ```toml
   [claude]
   command = [
     '{{if eq .Source "beads"}}sh{{end}}',
     '{{if eq .Source "beads"}}-c{{end}}',
     '''{{if eq .Source "beads"}}claude -p --permission-mode auto "/start {{.ID}}" && claude --permission-mode auto --continue -n "{{.Subject}}" /simplify{{end}}''',
   ]
   ```

2. Check the value loads, with a name no bead has: `work nonexistent-bead`. It refuses the bead, not the file the value came from.

3. Run `work <id>` on a ready ticket. `/start` streams into the terminal with no prompt box, and when its turn ends an interactive session opens titled `<id>: <title>`, carrying the turns `/start` already took.

## See also

- [`claude`](../references/claude.md#values) — what the command renders with
- [Configuration](../references/configuration.md#commands) — how the key is written
- [Command line](../references/cli.md#handoff) — how `work` reaches the command
