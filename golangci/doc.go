// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package golangci registers the analyzers of [go.dokimi.dev/lint/analysis] with golangci-lint, as
// the module plugins errorprefix and skipexpiry.
//
// golangci-lint runs a module plugin from a binary that its command custom builds. The command
// reads the package of the plugins from .custom-gcl.yml:
//
//	version: v2.14.0
//	plugins:
//	  - module: go.dokimi.dev/lint/golangci
//	    version: v0.2.0
//
// golangci-lint custom builds the binary custom-gcl from it. A .golangci.yml enables the linters:
//
//	version: "2"
//	linters:
//	  enable:
//	    - errorprefix
//	    - skipexpiry
//	  settings:
//	    custom:
//	      errorprefix:
//	        type: module
//	      skipexpiry:
//	        type: module
//
// In a repository of ergon, the key go.lint.plugins of .ergon.yaml lists each linter with the
// package and its version. ergon tool run builds the binary from that key, and ergon init renders
// the linters into .golangci.yml.
//
// Both linters run without settings. Each analyzer reads every package that golangci-lint loads,
// and the exclusions of .golangci.yml select the files whose reports golangci-lint shows, as they
// do for every linter.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/lint/analysis], register of
// github.com/golangci/plugin-module-register, and analysis of golang.org/x/tools.
package golangci
