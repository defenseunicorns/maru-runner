// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

// Package runner provides functions for running tasks in a tasks.yaml
package runner

import (
	"fmt"
	"testing"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/stretchr/testify/require"
)

func TestActionCmdMutation(t *testing.T) {
	// Initialize test cases
	testCases := []struct {
		input    string
		expected string
		config   string
	}{
		{
			input:    "./uds mycommand",
			expected: "/path/to/executable mycommand",
			config:   "uds",
		},
		{
			input:    "./uds ../uds/mycommand",
			expected: "/path/to/executable ../uds/mycommand",
			config:   "uds",
		},
		{
			input:    "./run ../run/mycommand",
			expected: "/path/to/executable ../run/mycommand",
			config:   "",
		},
	}

	// Run tests
	runCmd := "/path/to/executable"
	for _, tc := range testCases {
		config.CmdPrefix = tc.config
		t.Run(fmt.Sprintf("Input: %s", tc.input), func(t *testing.T) {
			mutatedCmd, err := actionCmdMutation(tc.input, runCmd)
			require.NoError(t, err)
			require.Equal(t, tc.expected, mutatedCmd)
		})
	}
}
