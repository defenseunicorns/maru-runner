// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

// Package config contains configuration strings for maru
package config

import (
	"runtime"
)

const (
	// TasksYAML is the string for the default tasks.yaml
	TasksYAML = "tasks.yaml"
)

var (
	// CLIArch is the computer architecture of the device executing the CLI commands
	CLIArch string

	// CLIVersion track the version of the CLI
	CLIVersion = "unset"

	// CmdPrefix is used to prefix Zarf cmds (like wait-for), useful when vendoring both the runner and Zarf
	// if not set, the system Zarf will be used
	CmdPrefix string

	// EnvPrefix is the prefix for viper configs and runner variables, useful when vendoring the runner
	EnvPrefix = "run"

	// ListTasks is a flag to print available tasks in a TaskFileLocation
	ListTasks bool

	// LogLevel is the log level for the runner
	LogLevel string

	// TaskFileLocation is the location of the tasks file to run
	TaskFileLocation string

	// SetRunnerVariables is a map of the run time variables defined using --set
	SetRunnerVariables map[string]string

	// SkipLogFile is a flag to skip logging to a file
	SkipLogFile bool

	// TempDirectory is the directory to store temporary files
	TempDirectory string
)

// GetArch returns the arch based on a priority list with options for overriding.
func GetArch(archs ...string) string {
	// List of architecture overrides.
	priority := append([]string{CLIArch}, archs...)

	// Find the first architecture that is specified.
	for _, arch := range priority {
		if arch != "" {
			return arch
		}
	}

	return runtime.GOARCH
}
