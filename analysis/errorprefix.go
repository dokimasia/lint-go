// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package analysis

import (
	"go/ast"
	"go/constant"
	"go/types"
	"path"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

// errorsNew is the full name of the function whose text errorprefix checks: New of the package
// errors of the standard library.
const errorsNew = "errors.New"

// The messages of a finding of errorprefix: the text and the prefix that it lacks, in a library and
// in a command.
const (
	prefixMessage  = "the text %q of errors.New does not start with %q, the name of its package"
	commandMessage = "the text %q of errors.New does not start with %q, the name of its command"
)

// ErrorPrefix returns the analyzer errorprefix. It reports each call of errors.New, under any name
// of the import of errors, in a file other than a test file, whose text is a constant that does not
// start with the name of its package and a colon, such as clock: instant is zero, or with the name
// and a dot, for a qualifier such as kernel.patch: malformed. In a package main the name is the name
// of the command, as go build names its binary: the last element of the import path, such as greet
// for example.com/greet/cmd/greet, or the element before it when the last element is a major
// version, such as tool for example.com/tool/v2. A text that is no constant, such as a variable, is
// not reported. The analyzer skips a generated file and every package that skip reports.
//
// The text of an error that starts with the name of its package states where the error comes from
// when a caller far from that package reports it. A user of a command reads the name of the command
// on standard error.
func ErrorPrefix(skip Skip) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "errorprefix",
		Doc:  "report each errors.New whose text does not start with the name of its package",
		URL:  "https://pkg.go.dev/go.dokimi.dev/lint/analysis#ErrorPrefix",
		Run: func(pass *analysis.Pass) (any, error) {
			name, message := prefix(pass.Pkg)
			for _, f := range scope(pass, skip) {
				if isTest(pass, f) {
					continue
				}
				for n := range ast.Preorder(f) {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						continue
					}
					fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
					if !ok || fn.FullName() != errorsNew {
						continue
					}
					text := pass.TypesInfo.Types[call.Args[0]].Value
					if text == nil {
						continue
					}
					s := constant.StringVal(text)
					if !strings.HasPrefix(s, name+":") && !strings.HasPrefix(s, name+".") {
						pass.Reportf(call.Pos(), message, s, name+":")
					}
				}
			}
			return nil, nil
		},
	}
}

// prefix returns the name that each error text of pkg starts with, and the message of a finding.
// A package main has the name of its command. A major version is an element v2, v3 and so on, but
// not v0, v1 or an element with a leading zero such as v05.
func prefix(pkg *types.Package) (string, string) {
	if pkg.Name() != "main" {
		return pkg.Name(), prefixMessage
	}
	dir, elem := path.Split(pkg.Path())
	major := len(elem) >= 2 && elem[0] == 'v' && elem[1] != '0' && elem != "v1" &&
		strings.Trim(elem[1:], "0123456789") == ""
	if dir != "" && major {
		elem = path.Base(dir)
	}
	return elem, commandMessage
}
