// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
)

// childVar is the variable of the environment that makes the test binary run main in place of its
// tests. A test that runs the binary again sets it, so main runs in a process whose coverage go
// test merges into the profile.
const childVar = "DOKIMI_LINT_GO_MAIN"

// The findings of the module of the cases, each after the path of its file in the module, which
// dokimi-lint-go prints with the separator of the system.
var (
	demoPrefix = filepath.FromSlash("demo/demo.go") +
		`:6:14: the text "bad" of errors.New does not start with "demo:", the name of its package`
	demoSkip = filepath.FromSlash("demo/demo_test.go") +
		":6:2: the skip expired on 2020-01-01, so fix the test and remove the skip"
	legacy = filepath.FromSlash("legacy/legacy.go") +
		`:6:14: the text "old" of errors.New does not start with "legacy:"`
	lockedSend = filepath.FromSlash("locked/locked.go") +
		":18:7: a send to a channel blocks inside the critical section of s.mu from line 16"
	lockedSync = filepath.FromSlash("locked/locked.go") +
		":19:15: locked.Device.Sync of a blocking interface blocks inside the critical section of s.mu from line 16"
)

// locked is the package of the cases that sends and calls Sync of a Device while a lock is held.
const locked = `package locked

import "sync"

// Device is an interface whose methods block.
type Device interface{ Sync() error }

// Store has a lock and a channel.
type Store struct {
	mu sync.Mutex
	ch chan int
}

// Put sends and syncs d while mu is locked.
func (s *Store) Put(d Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ch <- 1
	return d.Sync()
}
`

// module is the module of the cases: the package demo breaks the rules of errorprefix and
// skipexpiry, the package legacy breaks the rule of errorprefix, and the package locked breaks the
// rule of lockblock.
var module = files.Tree{
	"go.mod": files.Text("module example.com/demo\n\ngo 1.27\n"),
	"demo/demo.go": files.Text(
		"package demo\n\nimport \"errors\"\n\n// ErrBad has no prefix.\nvar ErrBad = errors.New(\"bad\")\n",
	),
	"demo/demo_test.go": files.Text(
		"package demo\n\nimport \"testing\"\n\nfunc TestOld(t *testing.T) {\n\tt.Skip(\"expires 2020-01-01\")\n}\n",
	),
	"legacy/legacy.go": files.Text(
		"package legacy\n\nimport \"errors\"\n\n// ErrOld has no prefix.\nvar ErrOld = errors.New(\"old\")\n",
	),
	"locked/locked.go": files.Text(locked),
}

func TestMain(m *testing.M) {
	if os.Getenv(childVar) != "" {
		main()
	}
	os.Exit(m.Run())
}

// TestDokimiLintGoProcess runs the test binary as dokimi-lint-go, one process at a time, because
// two processes that exit in the same nanosecond write one file of coverage.
func TestDokimiLintGoProcess(t *testing.T) {
	t.Run("main", func(t *testing.T) {
		tests := []struct {
			name    string
			args    []string
			env     []string
			status  int
			reports []string
			omits   []string
		}{
			{
				name:    "exits 3 with the findings of every analyzer",
				args:    []string{"./..."},
				status:  3,
				reports: []string{demoPrefix, demoSkip, legacy, lockedSend},
				omits:   []string{lockedSync},
			},
			{
				name:    "skips the packages of each pattern of -exclude",
				args:    []string{"-exclude=./legacy/...", "./..."},
				status:  3,
				reports: []string{demoPrefix, demoSkip},
				omits:   []string{legacy},
			},
			{
				name:    "reports the calls through the interfaces of -lockblock.interfaces",
				args:    []string{"-lockblock.interfaces=example.com/demo/locked.Device", "./locked"},
				status:  3,
				reports: []string{lockedSend, lockedSync},
			},
			{
				name:    "runs the other analyzers without lockblock under -lockblock=false",
				args:    []string{"-lockblock=false", "./..."},
				status:  3,
				reports: []string{demoPrefix, demoSkip, legacy},
				omits:   []string{lockedSend},
			},
			{
				name:   "exits 0 for packages without a finding",
				args:   []string{"-exclude=./legacy/...", "-exclude=./demo", "-exclude=./locked", "./..."},
				status: 0,
				omits:  []string{demoPrefix, demoSkip, legacy, lockedSend},
			},
			{
				name:    "exits 2 for a pattern of -exclude that go list does not resolve",
				args:    []string{"-exclude=./missing", "./..."},
				status:  2,
				reports: []string{`invalid value "./missing" for flag -exclude: `, "-exclude pattern"},
			},
			{
				name:    "exits 2 for a pattern of -exclude without the go command",
				args:    []string{"-exclude=./legacy/...", "./..."},
				env:     []string{"PATH="},
				status:  2,
				reports: []string{`invalid value "./legacy/..." for flag -exclude: `},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				status, stderr := vet(t, files.Workspace(t, module), tt.env, tt.args...)
				assert.Equal(t, status, tt.status, "the exit status: "+stderr)
				for _, line := range tt.reports {
					assert.Contains(t, stderr, line, "the standard error")
				}
				for _, line := range tt.omits {
					assert.NotContains(t, stderr, line, "the standard error")
				}
			})
		}
	})
}

func TestExcluded(t *testing.T) {
	t.Parallel()

	t.Run("String", func(t *testing.T) {
		t.Parallel()

		t.Run("returns the import paths sorted and separated by commas", func(t *testing.T) {
			t.Parallel()
			e := excluded{"example.com/demo/legacy": true, "example.com/demo/demo": true}
			assert.Equal(t, e.String(), "example.com/demo/demo,example.com/demo/legacy", "the import paths")
		})
	})
}

// vet runs the test binary as dokimi-lint-go with args in the module dir, outside any workspace
// of Go, with the variables of env after those of the test, and returns its exit status and its
// standard error.
func vet(t *testing.T, dir string, env []string, args ...string) (int, string) {
	t.Helper()
	binary, err := os.Executable()
	assert.NoError(t, err, "the path of the test binary")
	cmd := exec.CommandContext(t.Context(), binary, args...)
	cmd.Dir = dir
	cmd.Env = append(append(os.Environ(), childVar+"=1", "GOWORK=off"), env...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	err = cmd.Run()
	if exit, ok := errors.AsType[*exec.ExitError](err); ok {
		return exit.ExitCode(), stderr.String()
	}
	assert.NoError(t, err, "the run of dokimi-lint-go")
	return 0, stderr.String()
}
