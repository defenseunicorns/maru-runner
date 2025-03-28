// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The Maru Authors

// Package test provides e2e tests for the runner.
package test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunnerInputs(t *testing.T) {
	t.Run("test that default values for inputs work when not required", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "has-default-empty", "--file", "src/test/tasks/inputs/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "default")
		require.NotContains(t, stdErr, "{{")
	})

	t.Run("test that default values for inputs work when required", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "has-default-and-required-empty", "--file", "src/test/tasks/inputs/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "default")
		require.NotContains(t, stdErr, "{{")

	})

	t.Run("test that default values for inputs work when required and have values supplied", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "has-default-and-required-supplied", "--file", "src/test/tasks/inputs/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "supplied-value")
		require.NotContains(t, stdErr, "{{")
	})

	t.Run("test that direct calling of task with default values for required inputs work", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "has-default-and-required", "--file", "src/test/tasks/inputs/tasks-with-inputs.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Completed \"echo $INPUT_HAS_DEFAULT_AND_REQUIRED; \"")
	})

	t.Run("test that direct calling of task without default values for required inputs fails", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "no-default-and-required", "--file", "src/test/tasks/inputs/tasks-with-inputs.yaml")
		require.Error(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Failed to run action: task \"no-default-and-required\" is missing required inputs:")
	})

	t.Run("test that inputs that aren't required with no default don't error", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "no-default-empty", "--file", "src/test/tasks/inputs/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.NotContains(t, stdErr, "has-no-default")
		require.NotContains(t, stdErr, "{{")
	})

	t.Run("test that inputs with no defaults that aren't required don't error when supplied with a value", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "no-default-supplied", "--file", "src/test/tasks/inputs/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "success + supplied-value")
		require.NotContains(t, stdErr, "{{")
	})

	t.Run("test that tasks that require inputs with no defaults error when called without values", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "no-default-and-required-empty", "--file", "src/test/tasks/inputs/tasks.yaml")
		require.Error(t, err, stdOut, stdErr)
		require.NotContains(t, stdErr, "{{")
	})

	t.Run("test that tasks that require inputs with no defaults run when supplied with a value", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "no-default-and-required-supplied", "--file", "src/test/tasks/inputs/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "supplied-value")
		require.NotContains(t, stdErr, "{{")
	})

	t.Run("test that when a task is called with extra inputs it warns", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "no-default-and-required-supplied-extra", "--file", "src/test/tasks/inputs/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "supplied-value")
		require.Contains(t, stdErr, "WARNING")
		require.Contains(t, stdErr, "does not have an input named extra")
		require.NotContains(t, stdErr, "{{")
	})

	t.Run("test that displays a deprecated message", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "deprecated-task", "--file", "src/test/tasks/inputs/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "WARNING")
		require.Contains(t, stdErr, "This input has been marked deprecated: This is a deprecated message")
	})

	t.Run("test that variables can be used as inputs", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "variable-as-input", "--file", "src/test/tasks/inputs/tasks.yaml", "--set", "foo=im a variable")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "im a variable")
	})

	t.Run("test that env vars can be used as inputs and take precedence over default vals", func(t *testing.T) {
		err := os.Setenv("MARU_FOO", "im an env var")
		require.NoError(t, err)
		stdOut, stdErr, runErr := e2e.Maru("run", "variable-as-input", "--file", "src/test/tasks/inputs/tasks.yaml")
		err = os.Unsetenv("MARU_FOO")
		require.NoError(t, err)
		require.NoError(t, runErr, stdOut, stdErr)
		require.Contains(t, stdErr, "im an env var")
	})

	t.Run("test that a --set var has the greatest precedence for inputs", func(t *testing.T) {
		err := os.Setenv("MARU_FOO", "im an env var")
		require.NoError(t, err)
		stdOut, stdErr, runErr := e2e.Maru("run", "variable-as-input", "--file", "src/test/tasks/inputs/tasks.yaml", "--set", "foo=most specific")
		err = os.Unsetenv("MARU_FOO")
		require.NoError(t, err)
		require.NoError(t, runErr, stdOut, stdErr)
		require.Contains(t, stdErr, "most specific")
	})

	t.Run("test that variables in directly called included tasks take the root default", func(t *testing.T) {
		stdOut, stdErr, err := e2e.Maru("run", "with:echo-foo", "--file", "src/test/tasks/inputs/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "default-value")
	})

	t.Run("test that variables in directly called included tasks take empty set values", func(t *testing.T) {
		stdOut, stdErr, err := e2e.Maru("run", "with:echo-foo", "--file", "src/test/tasks/inputs/tasks.yaml", "--set", "foo=''")
		require.NoError(t, err, stdOut, stdErr)
		require.NotContains(t, stdErr, "default-value")
	})

	t.Run("test that variables in directly called included tasks pass through even when not in the root", func(t *testing.T) {
		stdOut, stdErr, err := e2e.Maru("run", "with:echo-bar", "--file", "src/test/tasks/inputs/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "default-value")
	})

	t.Run("test that using the --with command line flag works", func(t *testing.T) {
		stdOut, stdErr, err := e2e.Maru("run", "with:command-line-with", "--file", "src/test/tasks/inputs/tasks.yaml", "--with", "input1=input1", "--with", "input3=notthedefault", "--set", "FOO=baz")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Input1Tmpl: input1 Input1Env: input1 Input2Tmpl: input2 Input2Env: input2 Input3Tmpl: notthedefault Input3Env: notthedefault Var: baz")
	})
}
