# Tests

## Coverage lost to the handoff and the stand-ins

The total reads below what the suite reaches. `go test -coverprofile` counts nothing a child ran once `syscall.Exec` replaced it, which is every case `internal/cli`'s harness types through `session.hands`, and nothing a stand-in runs.

A child that returns and exits is counted, because the coverage runtime flushes its counters from an exit hook. An exec never reaches that hook, and a stand-in answers from `internal/testenv`'s `init`, before the runtime installs one. Neither can be flushed by hand: `runtime/coverage.WriteCountersDir` refuses inside a test binary, and the `-covermode=atomic` its message asks for changes nothing.

Read a drop in the total against what moved out of the test binary, and do not restore an in-package test to lift the number.
