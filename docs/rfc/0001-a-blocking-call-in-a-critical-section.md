---
rfc: 0001
title: A blocking call in a critical section
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-11-13
updated: 2026-11-13
discussion: https://github.com/dokimasia/lint-go/issues/5
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# RFC-0001: A blocking call in a critical section

## Summary

A goroutine that keeps a `sync.Mutex` or a `sync.RWMutex` locked across a
blocking call makes every other caller of the lock wait for that call.
This proposal adds the analyzer `lockblock`. It reports each blocking
operation inside the critical section of a lock, and each call of a
function that makes one, with the path of calls to the operation.

- A critical section runs from a `Lock` or `RLock` to the `Unlock` or
  `RUnlock` of the same lock in the same function.
- A blocking operation is a send, a receive, a `select` without a
  `default`, `time.Sleep`, `sync.WaitGroup.Wait`, `sync.Cond.Wait` of
  another lock, or a call through an interface that a setting names.
- Each function has a summary: the path of calls to a blocking
  operation, and the number of locks that its callers hold when the
  operation blocks inside one of their critical sections. The analyzer
  exports the summary as a fact of `golang.org/x/tools/go/analysis`,
  which the analysis of each importing package reads.
- A function that unlocks a lock of its caller, as a wait for a
  condition does, does not block inside the critical section of that lock
  until it locks the lock again.

## Motivation

### Waits behind a lock

When the call under the lock is the sync of a disk that hangs, every
caller of the lock waits until the kernel returns, also the callers that
work on another disk behind the same lock. A bubble of
`testing/synctest` then never becomes quiescent, because Go counts a wait
for a mutex as not durable (golang/go#77687).

An audit of a storage engine found 28 of its 62 mutexes locked across
such a call. The audit built the SSA form of 43 production packages with
a call graph of variable type analysis. It followed each `Lock` to its
unlock in the same function, or to the end of the function after a
deferred unlock, and reported each blocking call between the two.

### No analyzer reports it

vet's `copylocks` reports a copy of a lock. No analyzer of vet or of
golangci-lint reports a call under one, so only a review checks the rule
today.

## Detailed design

### Components

| Component | Package | Responsibility |
|---|---|---|
| `LockBlock` | `analysis` | Returns the analyzer `lockblock`, with the flags `interfaces` and `hooks` |
| The summaries | `analysis` | The blocking operation of each function, the number of locks of the callers that it needs, and the fact that exports both |
| The critical sections | `analysis` | The locks that every path holds at each instruction of a function, and the locks of the callers that a path unlocks |
| `dokimi-lint-go` | `cmd/dokimi-lint-go` | Runs `lockblock` with the other analyzers, and exposes its flags as `-lockblock.interfaces` and `-lockblock.hooks` |
| The plugin `lockblock` | `golangci` | Registers the analyzer for golangci-lint, and decodes its settings |

### The critical section

The analyzer reads the SSA form of each function, which the analyzer
`buildssa` builds.

- A critical section begins at a call of `Lock` or `RLock` of a
  `sync.Mutex` or a `sync.RWMutex`, also through a promoted method of an
  embedded lock.
- It ends at the call of `Unlock` or `RUnlock` of the same lock. A
  deferred `Unlock` or `RUnlock` ends it when the function returns, so the
  rest of the function is inside it.
- Calls lock the same lock when their receivers have the same access
  path: the same variable, parameter, free variable or global, then the
  same fields, dereferences and index values.
- An instruction is inside the critical section when every path to it
  holds the lock. A lock that one path holds and another does not is not
  held there.
- `TryLock` and `TryRLock` begin no critical section, because the lock is
  held on one result alone.
- A call of `Lock` through a method value, such as `lock := mu.Lock`,
  begins no critical section. SSA keeps the receiver of a method value
  inside its wrapper, so the call has no receiver to compare.

### A lock of the caller

A function such as a wait for a condition unlocks a lock that its caller
locked, blocks, and locks the lock again before it returns.

- An `Unlock` or `RUnlock` unlocks a lock of the callers when the
  function has not locked or tried that lock on any path to the call.
- A later `Lock` or `RLock` of that lock locks it again for the callers.
  The function does not hold it as a lock of its own.
- A blocking operation between the two does not block inside the
  critical section of that lock.
- The analyzer assumes that a function unlocks the locks that its caller
  locked last. A count of locks then states the effect of a call, and a
  summary does not need the access path of a lock of the caller.

### Blocking operations

| Operation | Blocks |
|---|---|
| A send to a channel | Always |
| A receive from a channel, also in a `range` over a channel | Always |
| A `select` without a `default` case | Always |
| A call of `time.Sleep` or `sync.WaitGroup.Wait` | Always |
| A call of `sync.Cond.Wait` | When two or more locks are held at it, counting the locks that the callers hold |
| A call through an interface that the setting `interfaces` names | Always |
| A call through an interface that the setting `hooks` names | Always, reported as a hook |
| A call of a function whose summary blocks | Always |

`sync.Cond.Wait` releases the lock of its cond while it waits, and its
contract requires that lock to be held. So a `Wait` blocks a critical
section only when another lock is held as well. With one lock held, that
lock is the lock of the cond.

A receive from a timer, such as `<-time.After(d)` or `<-t.C`, is a receive
from a channel.

### Summaries and facts

A function blocks when its body makes a blocking operation, or calls a
function that blocks. Its summary states the path of calls to a blocking
operation, and the number of locks that the callers hold at the call when
the operation blocks inside one of their critical sections.

- A blocking operation needs one lock, and a `sync.Cond.Wait` needs two.
- At an operation that needs n locks, a function holds h locks of its own
  and has unlocked r locks of its callers. The operation then needs
  r + max(1, n − h) locks of the callers. A call of a function whose
  summary needs n locks counts the same way.
- The summary takes the operation that needs the fewest locks of the
  callers. A summary changes only to one that needs fewer locks, so the
  summaries of recursive functions settle.
- The analyzer computes the summary of every function of a package, its
  function literals included, until no summary changes.
- It exports the summary of each declared function and method as a fact,
  so the analysis of an importing package reads it. The analysis of a
  package runs over every package that it imports, as vet runs an
  analyzer with facts. A package `main` and a test file have no importer,
  so their functions do not export a fact.

| Body of the function | Locks of its own at the operation | Locks of the callers that it unlocked | Locks that the callers hold |
|---|---|---|---|
| `s.ch <- 1` | 0 | 0 | 1 |
| `s.mu.Lock(); s.cond.Wait()` | 1 | 0 | 1 |
| `s.cond.Wait()` | 0 | 0 | 2 |
| `s.mu.Unlock(); <-s.ch; s.mu.Lock()` | 0 | 1 | 2 |
| `s.mu.Unlock(); s.rw.Lock(); s.ch <- 1` | 1 | 1 | 2 |

A call follows the callee that SSA knows statically: a function, a method
of a concrete type, a function literal, and a method value of a concrete
type. These calls are not followed:

- A call through an interface that the settings do not name.
- A call of a function value, such as a parameter or a field.
- A call in a `go` statement, because the goroutine runs on its own.
- A deferred call.
- A call into the runtime or into an internal package of the standard
  library. Their channel operations belong to the scheduler and the
  collector, which every allocation can enter.

### Settings

| Setting | Flag of `dokimi-lint-go` | Value |
|---|---|---|
| `interfaces` | `-lockblock.interfaces` | The interfaces whose methods block, such as `io.Writer` or `example.com/disk.Device` |
| `hooks` | `-lockblock.hooks` | The interfaces whose methods run code outside the module, such as the hook of a plugin |

Each value is a list that commas separate. An entry is the import path of
the package, a dot, and the name of the interface. A call matches an entry
through the interface that declares the method, so a call of `Write`
through `io.ReadWriter` matches `io.Writer`.

- golangci-lint passes the same settings under
  `linters.settings.custom.lockblock.settings`. A `nolint` directive
  suppresses a report at its line.
- `dokimi-lint-go -lockblock=false` runs the other analyzers without
  `lockblock`, because multichecker gives each analyzer a flag of its
  name.

### The report

The analyzer reports each blocking operation and each blocking call once,
at its position, when the function holds at least the locks that the
operation or the callee needs. The report states the lock that the
function locked last of the locks that are still held during the
operation. With a callee that needs n locks, that lock is the n-th from
the last of the locks held there:

```text
store.go:57:2: a receive from a channel blocks inside the critical section of s.mu from line 42
store.go:61:9: the call of (*store).flush blocks inside the critical section of s.mu from line 42: (*store).flush -> (*disk).sync -> disk.Device.Sync of a blocking interface
```

The lock is the receiver of the `Lock` call as the source writes it, or
the first argument of a method expression such as
`(*sync.Mutex).Lock(mu)`.

### Scope

The analyzer reports in the files other than test files, and skips the
packages that its `Skip` reports, as `errorprefix` does. It reports
nothing in a generated file, and a function of a generated file still has
a summary.

### Invariants

- The lock of a report is held on every path to the reported position.
- The path of a summary consists of calls that SSA resolves statically,
  from the function to a blocking operation.
- A summary that needs n locks blocks inside a critical section of a
  caller that holds n locks, when each function on its path unlocks the
  locks that its caller locked last.
- The analysis of a package reads only the facts of the packages that it
  imports, so it runs under vet, `dokimi-lint-go` and golangci-lint.

### Limits

| Case | Result |
|---|---|
| A lock that one function takes and another releases | Not reported |
| A function that unlocks a lock of its caller and returns without locking it again | The caller still holds the lock after the call |
| A function that unlocks a lock other than the lock that its caller locked last | The report states the lock that the caller locked last of the locks that are still held |
| A callee that returns before its blocking operation for the arguments of a call | Reported, because a summary does not depend on the arguments |
| A lock through the interface `sync.Locker` | Not tracked |
| A wait inside the function of a `sync.Once` | Not reported, because the call of the function is dynamic |
| A send to a buffered channel that has room | Reported, because the capacity is a value at run time |

## Alternatives considered

### A. A call graph of the whole program

The audit used a call graph of variable type analysis over every package
at once, which resolves calls through interfaces and function values.

**Why not:** vet, `dokimi-lint-go` and golangci-lint run an analyzer one
package at a time, with facts from its imports. A whole-program analysis
needs every package at once, so it runs as neither. The settings name
the interfaces whose calls block instead.

### B. A lock that some path holds

The analyzer would report a call where any path to it holds the lock.

**Why not:** the lock of a report must be held when the call runs.

### C. Annotations on functions

Comments would mark the functions and the methods that block, as the
attributes of Clang's thread safety analysis mark the functions that
acquire a lock.

**Why not:** every package would need its annotations before the first
report. The summaries find the blocking functions themselves.

### D. Detection at run time

A debug build would check each blocking call against the locks that the
goroutine holds, as Linux does with `might_sleep()`.

**Why not:** a check at run time finds only the paths that the tests run,
and the analyzer reports every path.

### E. The identity of an unlocked lock

A summary would state the access path of each lock of the callers that
a function unlocks, from the parameters of the function. Clang's
`RELEASE` attribute states the same for a function that releases a lock.
A call would map each access path through its arguments to a lock of the
caller.

**Why not:** a summary would need a set of access paths for each
operation, and a closure or a global would need a mapping of its own. In
a run over the storage engine of the audit, 66 of 76 reports came from
three functions that unlock the lock of their receiver before they wait.
Each caller of the three locked that lock last, so the count removes the
66 reports, as the access paths would.

## Drawbacks

- The analysis of a package runs over every package that it imports, the
  standard library included, which costs the time of the SSA form of
  each.
- `dokimi-lint-go` suppresses a report only by its package, through
  `-exclude`, or turns the analyzer off with `-lockblock=false`.
  golangci-lint suppresses one at its line with `nolint`.
- A send to a buffered channel that never fills, and a receive from a
  channel that is always ready, are reported.
- In the run over the storage engine, each of the 10 reports that remain
  needs a `nolint` directive: six sends to a channel with room for its one
  value, and four calls whose argument makes the callee return before it
  receives.

## References

| What | Where |
|---|---|
| The issue | <https://github.com/dokimasia/lint-go/issues/5> |
| A wait for a mutex is not durable in a bubble | <https://github.com/golang/go/issues/77687> |
| Clippy's `await_holding_lock` | <https://rust-lang.github.io/rust-clippy/master/index.html#await_holding_lock> |
| Clang's thread safety analysis | <https://clang.llvm.org/docs/ThreadSafetyAnalysis.html> |
| Facts of `golang.org/x/tools/go/analysis` | <https://pkg.go.dev/golang.org/x/tools/go/analysis#hdr-Modular_analysis_with_Facts> |
