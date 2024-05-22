// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package config contains configuration strings for maru
package config

import (
	"runtime"
)

const (
	// TasksYAML is the string for the default tasks.yaml
	TasksYAML = "tasks.yaml"

	// EnvPrefix is the prefix for environment variables
	EnvPrefix = "MARU"
)

var (
	// CLIArch is the computer architecture of the device executing the CLI commands
	CLIArch string

	// CLIVersion track the version of the CLI
	CLIVersion = "unset"

	// CmdPrefix is used to prefix Zarf cmds (like wait-for), useful when vendoring both the runner and Zarf
	// if not set, the system Zarf will be used
	CmdPrefix string

	// TaskFileLocation is the location of the tasks file to run
	TaskFileLocation string

	// TempDirectory is the directory to store temporary files
	TempDirectory string

	extraEnv = map[string]string{"MARU": "true", "MARU_ARCH": GetArch()}
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

// AddExtraEnv adds a new envirmentment variable to the extraEnv to make it available to actions
func AddExtraEnv(key string, value string) {
	if extraEnv == nil {
		extraEnv = make(map[string]string)
	}
	extraEnv[key] = value
}

// GetExtraEnv returns the map of extra environment variables that have been set and made available to actions
func GetExtraEnv() map[string]string {
	return extraEnv
}

// ClearExtraEnv clears extraEnv back to empty map
func ClearExtraEnv() {
	extraEnv = make(map[string]string)
}
