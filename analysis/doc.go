// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package analysis is the analyzers of dokimi-lint-go, which lint-go of the baseline of ergon init
// runs in every module of Go: [ErrorPrefix] and [SkipExpiry].
//
//   - errorprefix reports a text of errors.New that does not start with the name of its package.
//   - skipexpiry reports a skipped test whose message states an expiry that has passed.
//
// Each is an analyzer of golang.org/x/tools/go/analysis, so dokimi-lint-go runs both as go vet
// runs its analyzers, over the packages of its arguments with their tests.
//
// # Scope
//
// Both analyzers skip a file that a tool generated, as [go/ast.IsGenerated] reports it, and every
// package that their [Skip] reports, with its external test package. dokimi-lint-go builds the
// Skip from the patterns of its flag -exclude, and lint-go of ergon passes the patterns of the key
// lint.exclude of the section go. errorprefix reads the files other than the test files, and
// skipexpiry the test files.
//
// # Dependency position
//
// Imports the standard library, golang.org/x/tools/go/analysis and
// golang.org/x/tools/go/types/typeutil. cmd/dokimi-lint-go and golangci import it.
package analysis
