// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Command dokimi-lint-go runs the analyzers errorprefix, skipexpiry and lockblock of
// [go.dokimi.dev/lint/analysis] over the packages of its arguments and their tests, as go vet runs
// its analyzers:
//
//	dokimi-lint-go -exclude=./internal/legacy/... ./...
//
// The flag -exclude takes a package pattern of go list, such as ./internal/legacy/..., and the
// analyzers skip each package that it matches with its external test package. The command resolves
// the pattern with go list while it parses its flags.
//
// The flags -lockblock.interfaces and -lockblock.hooks set the interfaces whose methods lockblock
// counts as blocking calls and as hooks, such as -lockblock.interfaces=io.Writer. Each analyzer has
// a flag of its name, so -lockblock=false runs the other analyzers without lockblock.
//
// The command exits as go vet does: with 0 for packages without a finding, 3 when an analyzer
// reports a finding, 1 for an error, such as a package that does not compile, and 2 for a command
// line that it refuses, such as a pattern of -exclude that go list does not resolve.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/lint/analysis],
// golang.org/x/tools/go/analysis/multichecker and golang.org/x/tools/go/packages.
package main
