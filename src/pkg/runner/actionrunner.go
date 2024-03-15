package runner

import (
	"context"

	"github.com/defenseunicorns/zarf/src/pkg/message"
	zarfUtils "github.com/defenseunicorns/zarf/src/pkg/utils"
	zarfTypes "github.com/defenseunicorns/zarf/src/types"
)

// ActionRunner defines the interface for running actions.
type ActionRunner interface {
	RunAction(ctx context.Context, cfg zarfTypes.ZarfComponentActionDefaults, cmd string,
		shellPref zarfTypes.ZarfComponentActionShell, spinner *message.Spinner) (string, error)
	GetConfig(cfg zarfTypes.ZarfComponentActionDefaults, a zarfTypes.ZarfComponentAction,
		vars map[string]*zarfUtils.TextTemplate) zarfTypes.ZarfComponentActionDefaults
}
