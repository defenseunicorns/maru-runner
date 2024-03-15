// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

package runner

import (
	"context"

	"github.com/defenseunicorns/zarf/src/pkg/message"
	zarfUtils "github.com/defenseunicorns/zarf/src/pkg/utils"
	zarfTypes "github.com/defenseunicorns/zarf/src/types"

	// allows us to use compile time directives
	_ "unsafe"
)

// zarfRunner is the default runner for zarf actions.
type zarfRunner struct {
}

// NewZarfRunner returns a new zarfRunner.
func NewZarfRunner() ActionRunner {
	return zarfRunner{}
}

func (z zarfRunner) RunAction(ctx context.Context, cfg zarfTypes.ZarfComponentActionDefaults, cmd string, shellPref zarfTypes.ZarfComponentActionShell, spinner *message.Spinner) (string, error) {
	return actionRun(ctx, cfg, cmd, shellPref, spinner)
}

func (z zarfRunner) GetConfig(cfg zarfTypes.ZarfComponentActionDefaults, a zarfTypes.ZarfComponentAction, vars map[string]*zarfUtils.TextTemplate) zarfTypes.ZarfComponentActionDefaults {
	return actionGetCfg(cfg, a, vars)
}

//go:linkname actionGetCfg github.com/defenseunicorns/zarf/src/pkg/packager.actionGetCfg
func actionGetCfg(cfg zarfTypes.ZarfComponentActionDefaults, a zarfTypes.ZarfComponentAction, vars map[string]*zarfUtils.TextTemplate) zarfTypes.ZarfComponentActionDefaults

//go:linkname actionRun github.com/defenseunicorns/zarf/src/pkg/packager.actionRun
func actionRun(ctx context.Context, cfg zarfTypes.ZarfComponentActionDefaults, cmd string, shellPref zarfTypes.ZarfComponentActionShell, spinner *message.Spinner) (string, error)
