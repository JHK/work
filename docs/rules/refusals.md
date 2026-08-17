# Refusals

Whose words a refusal is written in. What each verb refuses is [the command line](../references/cli.md).

## R5 — A precondition of work's own is refused in work's own words

A condition `work` states as its own is refused in work's own words: the refusal names neither the command work put to a tool nor the answer that tool came back with. These are work's own:

- a directory git reports no repository for
- a worktree carrying changes, with no `--force` given
- the main checkout, and the worktree the shell stands in
- a name no worktree could be made for
- a directory already sitting where a worktree would go

Every other refusal names the command as it was run, which is what a reader needs to put the question again by hand. That is every tool work asks through `internal/run`, and the conditions only git can judge, such as a name that passes work's naming rule and not git's.

*Enforced by:* `namesGit` in `internal/work/repo_test.go`, which the refusal tests of that package read each precondition against, and `asked` in `internal/run/run.go`, which builds the command line every other refusal carries. `TestOpenOutsideARepository` in `internal/work/candidates_test.go` asks the first bullet of a translated git too.
