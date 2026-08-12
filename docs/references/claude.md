# `claude`

A worktree is handed to the [command](configuration.md#commands) the `[claude]` keys name, `claude` by default. Beyond running it, `work` asks the agent one question: which conversations a worktree carries.

## Conversations

No `claude` invocation reports a directory's conversations without becoming one.

| Invocation | No conversation | One | Several |
|---|---|---|---|
| `claude` | starts one | starts another | starts another |
| `claude --continue` | `No conversation found to continue`, exit 1 | resumes it | resumes the newest, silently |
| `claude --resume` | a picker saying none were found | a picker of one, Enter still required | a picker, newest first |
| `claude --resume <id>` | — | resumes it | resumes that one |

An argument that is not a session id is a search term prefilling the picker's filter. An id no transcript carries is exit 1, as is dismissing the picker with Esc. A prompt may follow `--continue` or `--resume`, and is submitted into the conversation it restores.

`--name` names a conversation being started and renames one being resumed, reaching the transcript as `custom-title`, which is what the picker's row shows. A conversation named by neither `--name` nor `/name` falls back to the model's own title, then to its last prompt.

A resume carries the model the conversation ran under. `--permission-mode` is ignored alongside `--continue` and `--resume` both, and a conversation's own mode is not restored either.

Print mode (`claude -p`) writes a transcript like any other, marked `entrypoint: sdk-cli`. Neither `--continue` nor the picker offers it, and `--resume <id>` resumes it. A session given no prompt writes no transcript.

## The contract

`transcripts` in [internal/action/claude/](../../internal/action/claude/sessions.go) is the question: which conversations a worktree carries, newest first, less whatever the agent's own picker hides. Only a conversation's identifier is read off the answer, the list being the agent's own to draw.

`recorded` answers it off the transcript store. Both sit inside the [`claude` action](configuration.md#actions), not at a seam the core declares: a second agent is an action of its own and the commands naming its binary. `claude` is the only one written.
