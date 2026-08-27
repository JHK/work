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

## Tests

A test is a command as the user types it, run through the harness in [internal/cli/](internal/cli/) over the systems [internal/wiring](internal/wiring/) names. Read that harness first; no case builds its own. A test of an internal function earns its place only where no command reaches the behaviour, and says why in a line.

[internal/testenv](internal/testenv/) is the ground, and its doc comments say what it offers. A package whose tests reach git, the settings or the agent imports it and declares `func TestMain(m *testing.M) { testenv.Main(m) }`, held to [docs/rules/test-isolation.md](docs/rules/test-isolation.md). Nothing else enters the module for a test's sake, testify's `mock` and `suite` included: a stand-in is hand-written.

Assert a scalar or a refusal with testify's `require`, guarding [docs/rules/refusals.md](docs/rules/refusals.md); compare a slice, a map or a struct with `testenv.Equal(t, want, got, says)`, whose failure message is go-cmp's diff. Pass the sentence the case promises as the last argument, unless the assertion already reads as it.

Cases that differ only in their input are one function over a table, each row a `t.Run` named for what it is. Cases that establish different properties stay apart, however alike their setup, and none only re-walks a path another already holds. Coverage is read, not defended: no package carries a floor of its own.

A test file is named for the surface it covers. What two or more of a package's test files read lives in `ground_test.go` with `TestMain`; what one file reads stays in it, and no two stand-ins answer the same question. A test's name should carry what the case establishes; a comment above it only where the name cannot, in two lines at most.

[internal/cli/](internal/cli/) is the worked example of a command's tests, [internal/config/](internal/config/) of a package's own.
