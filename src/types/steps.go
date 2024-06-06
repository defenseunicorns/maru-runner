// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package types contains all the types used by the runner.
package types

import (
	"github.com/defenseunicorns/pkg/exec"
)

type Step struct {
	ID      string            `json:"id,omitempty" jsonschema:"description=A unique identifier for the step. You can use the id to reference the step in contexts."`
	Env     map[string]string `json:"env,omitempty" jsonschema:"description=Additional environment variables to set for the command"`
	WorkDir string            `json:"dir,omitempty" jsonschema:"description=The working directory to run the command in (default is CWD)"`

	Cmd    string                `json:"cmd,omitempty" jsonschema:"description=The command to run. Must specify cmd, script, or wait for the action to do anything."`
	Shell  *exec.ShellPreference `json:"shell,omitempty" jsonschema:"description=(cmd only) Indicates a preference for a shell for the provided cmd to be executed in on supported operating systems"`
	Script string                `json:"script,omitempty" jsonschema:"description=The script to run. Must specify cmd, script, or wait for the action to do anything."`
	Wait   *ActionWait           `json:"wait,omitempty" jsonschema:"description=Wait for a condition to be met before continuing. Must specify cmd, script, or wait for the action."`

	Uses string            `json:"uses,omitempty" jsonschema:"description=The task to run, mutually exclusive with cmd and wait"`
	With map[string]string `json:"with,omitempty" jsonschema:"description=Input parameters to pass to the task,type=object"`

	Timeout int `json:"timeout,omitempty" jsonschema:"description=Timeout in seconds for the command (default to 0, no timeout for cmd actions and 300, 5 minutes for wait actions)"`
	Retry   int `json:"retry,omitempty" jsonschema:"description=Retry the command if it fails up to given number of times (default 0)"`
}
