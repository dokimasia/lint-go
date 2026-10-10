<!--
  ~ Copyright Dokimasia B.V. 2026
  ~ SPDX-License-Identifier: MIT
-->

# lint-go

`go.dokimi.dev/lint` contains two analyzers of Go, the command that runs them, and the module plugins that run them in golangci-lint.

| Package | Contents |
|---|---|
| [`analysis`](analysis) | The analyzers `errorprefix` and `skipexpiry`, for `golang.org/x/tools/go/analysis` |
| [`cmd/dokimi-lint-go`](cmd/dokimi-lint-go) | The command that runs both analyzers, as `go vet` runs its analyzers |
| [`golangci`](golangci) | The module plugins `errorprefix` and `skipexpiry` of golangci-lint |

- `errorprefix` reports a constant text of `errors.New` that does not start with the name of its package and a colon, such as `clock: instant is zero`.
- `skipexpiry` reports a skipped test whose message states a day that has passed, such as `t.Skip("expires 2026-12-31")`.

Both analyzers skip generated files.

## Install

Each [release](https://github.com/dokimasia/lint-go/releases/latest) has archives of `dokimi-lint-go` for Linux, macOS and Windows on amd64 and arm64, deb, rpm and apk packages for Linux, and `checksums.txt` with the SHA-256 digest of each file. cosign signs `checksums.txt` without a key, and each archive and package has an SPDX SBOM. `gh attestation verify <file> --repo dokimasia/lint-go` checks the build provenance of a file.

With Homebrew, install the cask of the tap:

```sh
brew install --cask dokimasia/tap/dokimi-lint-go
```

With Go 1.27.2 or later, install the command from source:

```sh
go install go.dokimi.dev/lint/cmd/dokimi-lint-go@latest
```

## Usage

```sh
dokimi-lint-go -exclude=./internal/legacy/... ./...
```

The flag `-exclude` takes a package pattern of `go list`, once for each pattern. The analyzers skip each package that a pattern matches, with its external test package.

The command exits as `go vet` does: 0 for packages without a finding, 3 for a finding, 1 for an error such as a package that does not compile, and 2 for a command line that it refuses.

## golangci-lint

`golangci-lint custom` builds a binary of golangci-lint with the module plugins of `.custom-gcl.yml`:

```yaml
version: v2.14.0
plugins:
  - module: go.dokimi.dev/lint/golangci
    version: v0.2.0
```

It writes the binary `custom-gcl`, which runs the linters that `.golangci.yml` enables:

```yaml
version: "2"
linters:
  enable:
    - errorprefix
    - skipexpiry
  settings:
    custom:
      errorprefix:
        type: module
      skipexpiry:
        type: module
```

Both linters run without settings. The exclusions of `.golangci.yml` select the files whose reports golangci-lint shows, as they do for every linter. In a repository of ergon, the key `go.lint.plugins` of `.ergon.yaml` lists each linter with the package and its version. `ergon tool run` builds the binary from that key, and `ergon init` renders the linters into `.golangci.yml`.

## Development

`make check` runs the gate of the repository: the formatters, the linters and the tests of every
language. [CONTRIBUTING.md](CONTRIBUTING.md) describes how to propose a change.

## License

lint-go is licensed under MIT. See [LICENSE](LICENSE).
