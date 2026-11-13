---
"go.dokimi.dev/lint": minor
---

Add the analyzer `lockblock` to `dokimi-lint-go` and to the module plugins, which reports a blocking call inside the critical section of a `sync.Mutex` or a `sync.RWMutex`.
