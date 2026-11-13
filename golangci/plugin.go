// Copyright Dokimasia B.V. 2026
// SPDX-License-Identifier: MIT

package golangci

import (
	"fmt"
	"time"

	"github.com/golangci/plugin-module-register/register"
	"go.dokimi.dev/lint/analysis"
	goanalysis "golang.org/x/tools/go/analysis"
)

// The names under which golangci-lint finds the plugins, which are the names of their analyzers.
const (
	errorPrefixName = "errorprefix"
	skipExpiryName  = "skipexpiry"
	lockBlockName   = "lockblock"
)

// init registers NewErrorPrefix, NewSkipExpiry and NewLockBlock under the names of their analyzers,
// which golangci-lint looks up for the linters of those names in its configuration.
func init() {
	register.Plugin(errorPrefixName, NewErrorPrefix)
	register.Plugin(skipExpiryName, NewSkipExpiry)
	register.Plugin(lockBlockName, NewLockBlock)
}

// lockBlockSettings are the settings of the plugin lockblock, which set the flags of its analyzer
// of the same names.
type lockBlockSettings struct {
	// Interfaces are the interfaces whose methods block.
	Interfaces []string `json:"interfaces"`
	// Hooks are the interfaces whose methods run code outside the module.
	Hooks []string `json:"hooks"`
}

// plugin is a module plugin of one analyzer of go.dokimi.dev/lint/analysis.
type plugin struct {
	// analyzer is the analyzer of the plugin.
	analyzer *goanalysis.Analyzer
}

var _ register.LinterPlugin = plugin{}

// BuildAnalyzers returns the analyzer of p alone.
func (p plugin) BuildAnalyzers() ([]*goanalysis.Analyzer, error) {
	return []*goanalysis.Analyzer{p.analyzer}, nil
}

// GetLoadMode asks golangci-lint for the types of each package, which every analyzer reads.
func (plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

// NewErrorPrefix returns the plugin errorprefix, from the settings of the linter in the
// configuration of golangci-lint. Its analyzer reads every package. The analyzer takes no settings,
// so NewErrorPrefix returns an error for any setting.
func NewErrorPrefix(settings any) (register.LinterPlugin, error) {
	if err := noSettings(errorPrefixName, settings); err != nil {
		return nil, err
	}
	return plugin{analyzer: analysis.ErrorPrefix(func(string) bool { return false })}, nil
}

// NewSkipExpiry returns the plugin skipexpiry, from the settings of the linter in the configuration
// of golangci-lint. Its analyzer reads every package, and compares each expiry with the day of the
// clock of the system. The analyzer takes no settings, so NewSkipExpiry returns an error for any
// setting.
func NewSkipExpiry(settings any) (register.LinterPlugin, error) {
	if err := noSettings(skipExpiryName, settings); err != nil {
		return nil, err
	}
	return plugin{analyzer: analysis.SkipExpiry(time.Now, func(string) bool { return false })}, nil
}

// NewLockBlock returns the plugin lockblock, from the settings of the linter in the configuration of
// golangci-lint: the lists interfaces and hooks, which set the flags of the same names of its
// analyzer. Its analyzer reads every package. NewLockBlock returns an error for settings of another
// form, and for an entry that the analyzer refuses.
func NewLockBlock(settings any) (register.LinterPlugin, error) {
	s, err := register.DecodeSettings[lockBlockSettings](settings)
	if err != nil {
		return nil, fmt.Errorf("golangci: the settings of %s: %w", lockBlockName, err)
	}
	a := analysis.LockBlock(func(string) bool { return false })
	flags := []struct {
		name    string
		entries []string
	}{{name: "interfaces", entries: s.Interfaces}, {name: "hooks", entries: s.Hooks}}
	for _, f := range flags {
		for _, entry := range f.entries {
			if err := a.Flags.Set(f.name, entry); err != nil {
				return nil, fmt.Errorf("golangci: %w", err)
			}
		}
	}
	return plugin{analyzer: a}, nil
}

// noSettings returns an error for the settings of the linter name, from the configuration of
// golangci-lint, unless they are none or an empty map, because neither analyzer takes settings.
func noSettings(name string, settings any) error {
	if _, err := register.DecodeSettings[struct{}](settings); err != nil {
		return fmt.Errorf("golangci: %s takes no settings: %w", name, err)
	}
	return nil
}
