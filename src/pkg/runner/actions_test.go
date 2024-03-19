// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

// Package runner provides functions for running tasks in a tasks.yaml
package runner

import (
	"fmt"
	"testing"
)

// Mocking zarfUtils for testing purposes
type mockZarfUtils struct{}

func (m mockZarfUtils) GetFinalExecutablePath() (string, error) {
	return "/path/to/executable", nil
}

func TestActionCmdMutation(t *testing.T) {
	// Initialize test cases
	testCases := []struct {
		input     string
		expected  string
		cmdPrefix Config
	}{
		{
			input:    "./uds mycommand",
			expected: "/path/to/executable mycommand",
			cmdPrefix: Config{
				cmdPrefix: "uds",
			},
		},
		{
			input:    "./uds ../uds/mycommand",
			expected: "/path/to/executable ../uds/mycommand",
			cmdPrefix: Config{
				cmdPrefix: "uds",
			},
		},
		{
			input:     "./run ../run/mycommand",
			expected:  "/path/to/executable ../run/mycommand",
			cmdPrefix: Config{},
		},
		// Add more test cases as needed
	}

	// Run tests
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Input: %s", tc.input), func(t *testing.T) {
			mutatedCmd, err := actionCmdMutation(tc.input, mockZarfUtils{}, tc.cmdPrefix)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if mutatedCmd != tc.expected {
				t.Errorf("Expected mutated command: %s, got: %s", tc.expected, mutatedCmd)
			}
		})
	}
}
