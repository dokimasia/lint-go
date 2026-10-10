// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package golangci_test

import (
	"testing"

	"github.com/golangci/plugin-module-register/register"
	"go.dokimi.dev/assert"
	"go.dokimi.dev/lint/golangci"
	"golang.org/x/tools/go/analysis/analysistest"
)

// plugins are the plugins of the cases: the name under which golangci-lint finds each, the name and
// the function of its constructor, and the package of testdata whose reports its analyzer states.
var plugins = []struct {
	name        string
	constructor string
	build       register.NewPlugin
	fixture     string
}{
	{name: "errorprefix", constructor: "NewErrorPrefix", build: golangci.NewErrorPrefix, fixture: "errs"},
	{name: "skipexpiry", constructor: "NewSkipExpiry", build: golangci.NewSkipExpiry, fixture: "skips"},
}

func TestPlugin(t *testing.T) {
	t.Parallel()

	for _, p := range plugins {
		t.Run(p.constructor, func(t *testing.T) {
			t.Parallel()

			t.Run("is the constructor that golangci-lint finds under "+p.name, func(t *testing.T) {
				t.Parallel()
				build, err := register.GetPlugin(p.name)
				assert.NoError(t, err, "GetPlugin")
				got, err := build(nil)
				assert.NoError(t, err, "the constructor")
				analyzers, err := got.BuildAnalyzers()
				assert.NoError(t, err, "BuildAnalyzers")
				assert.Length(t, analyzers, 1, "the analyzers")
				assert.Equal(t, analyzers[0].Name, p.name, "the name of the analyzer")
			})

			accepted := []struct {
				name string
				give any
			}{
				{name: "returns a plugin for no settings", give: nil},
				{name: "returns a plugin for an empty map of settings", give: map[string]any{}},
			}
			for _, tt := range accepted {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := p.build(tt.give)
					assert.NoError(t, err, p.constructor)
					assert.NotNil(t, got, "the plugin")
				})
			}

			refused := []struct {
				name string
				give any
			}{
				{name: "returns an error for a setting", give: map[string]any{"exclude": []any{"./legacy/..."}}},
				{name: "returns an error for settings that are no map", give: "exclude"},
			}
			for _, tt := range refused {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					got, err := p.build(tt.give)
					assert.HasError(t, err, p.constructor)
					assert.Contains(t, err.Error(), "golangci: "+p.name+" takes no settings", "the error")
					assert.Nil(t, got, "the plugin")
				})
			}
		})
	}

	t.Run("BuildAnalyzers", func(t *testing.T) {
		t.Parallel()

		for _, p := range plugins {
			t.Run("returns an analyzer of "+p.name+" that skips no package", func(t *testing.T) {
				t.Parallel()
				got, err := p.build(nil)
				assert.NoError(t, err, p.constructor)
				analyzers, err := got.BuildAnalyzers()
				assert.NoError(t, err, "BuildAnalyzers")
				assert.Length(t, analyzers, 1, "the analyzers")
				analysistest.Run(t, analysistest.TestData(), analyzers[0], p.fixture)
			})
		}
	})

	t.Run("GetLoadMode", func(t *testing.T) {
		t.Parallel()

		for _, p := range plugins {
			t.Run("asks for the types of each package for "+p.name, func(t *testing.T) {
				t.Parallel()
				got, err := p.build(nil)
				assert.NoError(t, err, p.constructor)
				assert.Equal(t, got.GetLoadMode(), register.LoadModeTypesInfo, "the load mode")
			})
		}
	})
}
