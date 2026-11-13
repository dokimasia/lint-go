// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package analysis is the analyzers of dokimi-lint-go and of the module plugins of
// go.dokimi.dev/lint/golangci: [ErrorPrefix], [SkipExpiry] and [LockBlock].
//
//   - errorprefix reports a text of errors.New that does not start with the name of its package,
//     or with the name of its command in a package main.
//   - skipexpiry reports a skipped test whose message states an expiry that has passed.
//   - lockblock reports a blocking operation inside the critical section of a sync.Mutex or a
//     sync.RWMutex, and a call inside one of a function that blocks.
//
// Each is an analyzer of golang.org/x/tools/go/analysis. dokimi-lint-go runs them as go vet runs
// its analyzers, over the packages of its arguments with their tests, and golangci-lint runs them
// as module plugins.
//
// # Scope
//
// Every analyzer skips a file that a tool generated, as [go/ast.IsGenerated] reports it, and every
// package that its [Skip] reports, with its external test package. dokimi-lint-go builds the Skip
// from the patterns of its flag -exclude. errorprefix and lockblock read the files other than the
// test files, and skipexpiry the test files. lockblock summarizes the functions of a generated
// file too.
//
// # Dependency position
//
// Imports the standard library, golang.org/x/tools/go/analysis, its package passes/buildssa,
// golang.org/x/tools/go/ssa and golang.org/x/tools/go/types/typeutil. cmd/dokimi-lint-go and
// golangci import it.
package analysis
