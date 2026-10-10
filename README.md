<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# lint-go

`go.dokimi.dev/lint` contains two analyzers of Go and the command that runs them. The target `lint-go` of ergon's baseline runs the command in every module of a repository, beside golangci-lint.

| Package | Contents |
|---|---|
| [`analysis`](analysis) | The analyzers `errorprefix` and `skipexpiry`, for `golang.org/x/tools/go/analysis` |
| [`cmd/dokimi-lint-go`](cmd/dokimi-lint-go) | The command that runs both analyzers, as `go vet` runs its analyzers |

- `errorprefix` reports a constant text of `errors.New` that does not start with the name of its package and a colon, such as `clock: instant is zero`.
- `skipexpiry` reports a skipped test whose message states a day that has passed, such as `t.Skip("expires 2026-12-31")`.

Both analyzers skip generated files.

## Usage

```sh
go install go.dokimi.dev/lint/cmd/dokimi-lint-go@latest
dokimi-lint-go -exclude=./internal/legacy/... ./...
```

The flag `-exclude` takes a package pattern of `go list`, once for each pattern. The analyzers skip each package that a pattern matches, with its external test package. In a repository of ergon, the key `go.lint.exclude` of `.ergon.yaml` lists the patterns.

The command exits as `go vet` does: 0 for packages without a finding, 3 for a finding, 1 for an error such as a package that does not compile, and 2 for a command line that it refuses.

## Development

`make check` runs the gate of the repository: the formatters, the linters and the tests of every
language. [CONTRIBUTING.md](CONTRIBUTING.md) describes how to propose a change.

## License

lint-go is licensed under MIT. See [LICENSE](LICENSE).
