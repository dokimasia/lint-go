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
)

// init registers NewErrorPrefix and NewSkipExpiry under the names of their analyzers, which
// golangci-lint looks up for the linters of those names in its configuration.
func init() {
	register.Plugin(errorPrefixName, NewErrorPrefix)
	register.Plugin(skipExpiryName, NewSkipExpiry)
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

// GetLoadMode asks golangci-lint for the types of each package, which both analyzers read.
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

// noSettings returns an error for the settings of the linter name, from the configuration of
// golangci-lint, unless they are none or an empty map, because neither analyzer takes settings.
func noSettings(name string, settings any) error {
	if _, err := register.DecodeSettings[struct{}](settings); err != nil {
		return fmt.Errorf("golangci: %s takes no settings: %w", name, err)
	}
	return nil
}
