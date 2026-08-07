# The interactive launcher

*Epic: work-cli-39a*

## Intent

`work` is [bounded to one moment](../explanation/worktree-per-ticket.md): it resolves a target, hands the terminal over and exits. The screen is what a person meets in that moment, and where the choice of what to hand over belongs.

## Desired end state

Two questions stand between `work` and the handoff: what to work on, and what to hand it to. A `<name>` answers the first, a flag answers the second, and only what neither the arguments nor the [configuration](../references/configuration.md) answer reaches the screen. Each question that does is its own `fzf` invocation, one after the other.

The actions are agent, shell, editor, diff and ask. A worktree opens on one command, so the action is one value ([`work.Action`](../../internal/work/enter.go)) whatever named it: a flag, a configuration key, or the screen. The first four name a command the configuration defines; ask names the screen, and is the choice between the other four. The screen orders them and says which apply; it does not define what they run. Handing over is the last thing that happens, so no action asks anything of its own — except agent, which may.

Configuration answers the second question standing, once per moment: one key for the worktree just created, another for the one already there. The first is agent, the second is ask.

Agent is a single action however many conversations a worktree carries, and does the right thing for what it finds there: none starts one, one returns to it, several reach a list. That list is `claude`'s own, so the screen neither draws it nor names what is on it.

Two keys are what that costs, and [internal/sessions](../../internal/sessions/sessions.go) tells which one a worktree has earned before anything is handed over:

```toml
[agent]
start-session = ["claude", "--permission-mode", "auto", "--name={{.Name}}"]
resume-session = ["claude", "--resume", "{{.Session}}"]
```

`.Session` is the conversation where a worktree carries one alone, and empty where it carries several. An element rendering to nothing is dropped, so the one value either reaches that conversation or reaches the list, and no id is ever asked of a person: the launcher holds them already, having counted. `.Name` is the bead, the pull request or the branch, so a conversation started this way is the worktree's name in every later list; a resumed one keeps what it has, `--name` on it being a rename. `/start` is seeded where a worktree is created and nowhere else, so a worktree that has lost its conversation opens bare.

Nothing `claude` offers reports a directory's conversations without becoming one, so the count has no invocation behind it: `internal/sessions` reads the transcript store instead. It reports print-mode transcripts, which the picker hides, and filters them out before the count is read off it.

That reading is the whole of what an agent has to answer for, and it is one question — which conversations a worktree carries, newest first:

```go
// Conversations is what work needs of an agent beyond a command to run.
type Conversations interface {
	List(dir string) ([]Session, error)
}
```

`sessions.Claude` answers it off the transcript store; another agent answers it off whatever its own is, as a second implementation alongside. Only the identifier is read off a conversation, the list being the agent's own to draw. The agent is named rather than assumed, and a [how-to](../howto/) walks writing the next one: the interface to satisfy, where the code goes, and what makes it reachable. It is addressed to whoever brings that agent rather than to this repository, so it assumes nothing of `work` beyond the interface. `claude` stays the only one written.

Nothing is carried per worktree: `--model` and `--effort` go, and the configuration names a model and an effort where either is wanted at all. A resumed conversation brings its own model back regardless.

## What claude does

| Invocation | No conversation | One | Several |
|---|---|---|---|
| `claude` | starts one | starts another | starts another |
| `claude --continue` | `No conversation found to continue`, exit 1 | resumes it | resumes the newest, silently |
| `claude --resume` | a picker saying none were found | a picker of one, Enter still required | a picker, newest first |
| `claude --resume <id>` | — | resumes it | resumes that one |

An argument that is not a session id is a search term: it prefills the picker's filter, and a term matching one conversation still needs Enter. An id no transcript carries is exit 1. Dismissing the picker with Esc is exit 1, empty or not. A prompt may follow `--continue` or `--resume`, and is submitted into the restored conversation.

`--name` names a conversation being started and renames one being resumed, reaching the transcript as `custom-title`, which is what the picker's row shows. A conversation named by neither `--name` nor `/name` falls back to the model's own title, then to its last prompt.

A resume carries the model the conversation ran under. It does not carry the permission mode: `--permission-mode` is ignored alongside `--continue` and `--resume` both, and a conversation's own mode is not restored either.

Print mode (`claude -p`) writes a transcript like any other, marked `entrypoint: sdk-cli`. Neither `--continue` nor the picker offers it; `--resume <id>` resumes it, and `internal/sessions` lists it. A session given no prompt writes no transcript.
