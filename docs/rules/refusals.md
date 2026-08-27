# Refusals

Whose words a refusal is written in. What each verb refuses is [the command line](../references/cli.md).

## R5 — A precondition of work's own is refused in work's own words

A condition `work` states as its own is refused in work's own words: the refusal names neither the command work put to a tool nor the answer that tool came back with. These are work's own:

- a directory git reports no repository for
- a worktree carrying changes, with no `--force` given
- the main checkout, and the worktree the shell stands in
- a name no worktree could be made for
- a directory already sitting where a worktree would go
- a checkout carrying no changes, with `carry` given

Every other refusal names the command as it was run, which is what a reader needs to put the question again by hand. That is every tool work asks through `internal/run`, and the conditions only git can judge, such as a name that passes work's naming rule and not git's.

*Enforced by:* `result.came` in `internal/cli/ground_test.go`, which names a refusal whole beside every tool the run asked, so one of work's own is asserted saying nothing a tool said. Each precondition above is read that way in the same package: `repository_test.go` for the first, against a translated git too, `remove_test.go` for the next two, `move_test.go` and `add_test.go` for the two after them, and `carry_test.go` for the last. `CommandLine` in `internal/run/run.go` builds the line every other refusal carries.
