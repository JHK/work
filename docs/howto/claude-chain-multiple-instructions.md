# Claude: Chain multiple instructions

Run `/simplify` after `/start` in the session a ticket's new worktree opens on.

## Prerequisites

- A ticket title carrying no `$` or `"`: `{{.Title}}` is interpolated into a shell string.
- A decision on failure: `&&` skips the second instruction when the first exits non-zero, `;` in its place runs it either way.

## Steps

1. Set [`agent.start-ticket`](../references/configuration.md#keys) to the chain, wrapped in `sh -c`:

   ```toml
   [agent]
   start-ticket = [
     "sh", "-c",
     '''claude -p --permission-mode auto "/start {{.ID}}" && claude --permission-mode auto --continue -n "{{.ID}}: {{.Title}}" /simplify''',
   ]
   ```

2. Check the value loads, with a name no bead has: `work nonexistent-bead`. It refuses the bead, not the file the value came from.

3. Run `work <id>` on a ready ticket. `/start` streams into the terminal with no prompt box, and when its turn ends an interactive session opens titled `<id>: <title>`, carrying the turns `/start` already took.

## See also

- [Configuration](../references/configuration.md#commands) — what each key renders with, and the shipped default the chain replaces
- [Command line](../references/cli.md#handoff) — how `work` reaches the command
