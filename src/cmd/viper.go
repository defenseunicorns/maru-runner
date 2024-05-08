// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present the Maru Authors

// Package cmd contains the CLI commands for maru.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/config/lang"
	"github.com/defenseunicorns/zarf/src/pkg/message"
	"github.com/spf13/viper"
)

const (
	// Root config keys
	V_LOG_LEVEL    = "options.log_level"
	V_ARCHITECTURE = "options.architecture"
	V_NO_PROGRESS  = "options.no_progress"
	V_NO_LOG_FILE  = "options.no_log_file"
	V_TMP_DIR      = "options.tmp_dir"
	V_ENV_PREFIX   = "options.env_prefix"
)

var (
	// Viper instance used by the cmd package
	v *viper.Viper

	// holds any error from reading in Viper config
	vConfigError error
)

func initViper() {
	// Already initialized by some other command
	if v != nil {
		return
	}

	v = viper.New()

	// Specify an alternate config file
	cfgFile := os.Getenv("RUN_CONFIG")

	// Don't forget to read config either from cfgFile or from home directory!
	if cfgFile != "" {
		// Use config file from the flag.
		v.SetConfigFile(cfgFile)
	} else {
		// Search config paths (order matters!)
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.run")
		// todo: make configurable
		v.SetConfigName("run-config")
	}

	// we replace 'OPTIONS.' because in a run-config.yaml, the key is options.<opt>, but in the environment, it's RUN_<OPT>
	// e.g. RUN_LOG_LEVEL=debug
	v.SetEnvPrefix(config.EnvPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer("OPTIONS.", ""))
	v.AutomaticEnv()

	vConfigError = v.ReadInConfig()
	if vConfigError != nil {
		// Config file not found; ignore
		if _, ok := vConfigError.(viper.ConfigFileNotFoundError); !ok {
			message.WarnErr(vConfigError, fmt.Sprintf("%s - %s", lang.CmdViperErrLoadingConfigFile, vConfigError.Error()))
		}
	}
}

func printViperConfigUsed() {
	// Optional, so ignore file not found errors
	if vConfigError != nil {
		// Config file not found; ignore
		if _, ok := vConfigError.(viper.ConfigFileNotFoundError); !ok {
			message.WarnErr(vConfigError, fmt.Sprintf("%s - %s", lang.CmdViperErrLoadingConfigFile, vConfigError.Error()))
		}
	} else {
		message.Notef(lang.CmdViperInfoUsingConfigFile, v.ConfigFileUsed())
	}
}
