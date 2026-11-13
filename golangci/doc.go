// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

// Package golangci registers the analyzers of [go.dokimi.dev/lint/analysis] with golangci-lint, as
// the module plugins errorprefix, skipexpiry and lockblock.
//
// golangci-lint runs a module plugin from a binary that its command custom builds. The command
// reads the package of the plugins from .custom-gcl.yml:
//
//	version: v2.14.0
//	plugins:
//	  - module: go.dokimi.dev/lint/golangci
//	    version: v0.3.0
//
// golangci-lint custom builds the binary custom-gcl from it. A .golangci.yml enables the linters,
// and sets the settings of lockblock:
//
//	version: "2"
//	linters:
//	  enable:
//	    - errorprefix
//	    - skipexpiry
//	    - lockblock
//	  settings:
//	    custom:
//	      errorprefix:
//	        type: module
//	      skipexpiry:
//	        type: module
//	      lockblock:
//	        type: module
//	        settings:
//	          interfaces:
//	            - example.com/disk.Device
//	          hooks:
//	            - example.com/plugin.Hook
//
// In a repository of ergon, the key go.lint.analyzers of .ergon.yaml names the package with its
// version. ergon tool run builds the binary from that key, and ergon init enables errorprefix and
// skipexpiry in .golangci.yml.
//
// errorprefix and skipexpiry run without settings. lockblock takes the lists interfaces and hooks,
// which set the flags of the same names of its analyzer. Each analyzer reads every package that
// golangci-lint loads. The exclusions and the nolint directives of golangci-lint select the reports
// that it shows, as they do for every linter.
//
// # Dependency position
//
// Imports the standard library, [go.dokimi.dev/lint/analysis], register of
// github.com/golangci/plugin-module-register, and analysis of golang.org/x/tools.
package golangci
