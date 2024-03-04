// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

// Package lang contains the language strings in english used by maru
package lang

const (
	// root cmds
	RootCmdShort              = "CLI for the maru runner"
	RootCmdFlagSkipLogFile    = "Disable log file creation"
	RootCmdFlagLogLevel       = "Log level for the runner. Valid options are: warn, info, debug, trace"
	RootCmdErrInvalidLogLevel = "Invalid log level. Valid options are: warn, info, debug, trace."
	RootCmdFlagArch           = "Architecture for the runner"
	RootCmdFlagTempDir        = "Specify the temporary directory to use for intermediate files"

	// version
	CmdVersionShort = "Shows the version of the running runner binary"
	CmdVersionLong  = "Displays the version of the runner release that the current binary was built from."

	// internal
	CmdInternalShort             = "Internal cmds used by the runner"
	CmdInternalConfigSchemaShort = "Generates a JSON schema for the tasks.yaml configuration"
	CmdInternalConfigSchemaErr   = "Unable to generate the tasks.yaml schema"

	// cmd viper setup
	CmdViperErrLoadingConfigFile = "failed to load config file: %s"
	CmdViperInfoUsingConfigFile  = "Using config file %s"

	// run
	CmdRunFlag        = "Name and location of task file to run"
	CmdRunSetVarFlag  = "Set a runner variable from the command line (KEY=value)"
	CmdRunWithVarFlag = "Set the inputs for a task from the command line (KEY=value)"
	CmdRunList        = "List available tasks in a task file"
)
