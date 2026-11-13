// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package analysis_test

import (
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/lint/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestLockBlock(t *testing.T) {
	t.Parallel()

	t.Run("LockBlock", func(t *testing.T) {
		t.Parallel()

		t.Run("reports each blocking operation inside a critical section", func(t *testing.T) {
			t.Parallel()
			analysistest.Run(t, analysistest.TestData(), analysis.LockBlock(none), "locks")
		})

		t.Run("reports each call inside a critical section of a function that blocks", func(t *testing.T) {
			t.Parallel()
			analysistest.Run(t, analysistest.TestData(), analysis.LockBlock(none), "calls", "uses")
		})

		t.Run("reports each call of a method of an interface that its settings name", func(t *testing.T) {
			t.Parallel()
			a := analysis.LockBlock(none)
			assert.NoError(t, a.Flags.Set("interfaces", "ifaces.Device,other.Device"), "the blocking interfaces")
			assert.NoError(t, a.Flags.Set("hooks", "ifaces.Plugin"), "the hooks")
			analysistest.Run(t, analysistest.TestData(), a, "ifaces")
		})

		t.Run("reports nothing in a package that its skip reports", func(t *testing.T) {
			t.Parallel()
			a := analysis.LockBlock(func(path string) bool { return path == "quiet" })
			assert.NoError(t, a.Flags.Set("interfaces", "quiet.Device"), "the blocking interfaces")
			analysistest.Run(t, analysistest.TestData(), a, "quiet")
		})

		t.Run("reports no call of a method of an interface that no setting names", func(t *testing.T) {
			t.Parallel()
			a := analysis.LockBlock(none)
			assert.NoError(t, a.Flags.Set("interfaces", "other.Device"), "an interface of another package")
			analysistest.Run(t, analysistest.TestData(), a, "unnamed")
		})

		t.Run("states the interfaces and the hooks of its flags", func(t *testing.T) {
			t.Parallel()
			a := analysis.LockBlock(none)
			assert.NoError(t, a.Flags.Set("interfaces", "io.Writer,io.Reader"), "two interfaces")
			assert.NoError(t, a.Flags.Set("interfaces", "example.com/disk.Device"), "a third interface")
			assert.Equal(t, a.Flags.Lookup("interfaces").Value.String(), "example.com/disk.Device,io.Reader,io.Writer",
				"the interfaces in order")
			assert.Equal(t, a.Flags.Lookup("hooks").Value.String(), "", "no hook")
		})

		entries := []struct {
			name string
			give string
		}{
			{name: "refuses an entry without an import path", give: "Plugin"},
			{name: "refuses an entry that starts with a dot", give: ".Plugin"},
			{name: "refuses an entry without the name of an interface", give: "example.com/plugin."},
		}
		for _, tt := range entries {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := analysis.LockBlock(none).Flags.Set("hooks", tt.give)
				assert.HasError(t, err, "Set")
				want := `lockblock: the entry "` + tt.give + `" is no import path, a dot and an interface name`
				assert.Equal(t, err.Error(), want, "the entry")
			})
		}

		t.Run("is named lockblock", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, analysis.LockBlock(none).Name, "lockblock", "the name of the analyzer")
		})
	})
}
