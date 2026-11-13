# go.dokimi.dev/lint

## 0.3.0

### Minor Changes

- 0367daf: Add the analyzer `lockblock` to `dokimi-lint-go` and to the module plugins, which reports a blocking call inside the critical section of a `sync.Mutex` or a `sync.RWMutex`.

### Patch Changes

- cefe9a6: Fix `errorprefix` to expect the name of the command in a package `main`, such as `greet:`, in place of `main:`.

## 0.2.0

### Minor Changes

- e835f4e: Add the package `golangci`, which registers `errorprefix` and `skipexpiry` as module plugins of golangci-lint.

### Patch Changes

- c20a2e9: Publish `dokimi-lint-go` as a cask of the tap `dokimasia/homebrew-tap`.

## 0.1.0

### Minor Changes

- 27e0252: Add the analyzers errorprefix and skipexpiry, and the command dokimi-lint-go that runs them, from ergon-go-vet of go.dokimi.dev/ergon/lang/go.
