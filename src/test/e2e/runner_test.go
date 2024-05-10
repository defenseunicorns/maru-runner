// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The Maru Authors

// Package test provides e2e tests for the runner.
package test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskRunner(t *testing.T) {
	t.Log("E2E: Task Maru")

	t.Run("run action", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "action", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "specific test string")
	})

	t.Run("run cmd-set-variable", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "cmd-set-variable", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "I'm set from setVariables - unique-value")
		require.Contains(t, stdErr, "I'm set from a runner var - replaced")
	})
	t.Run("run default task", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "This is the default task")
	})

	t.Run("run default task when undefined", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "--file", "src/test/tasks/tasks-no-default.yaml")
		require.Error(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "task name default not found")
	})

	t.Run("run reference", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "reference", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "other-task")
	})

	t.Run("run recursive", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "recursive", "--file", "src/test/tasks/tasks.yaml")
		require.Error(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "task loop detected")
	})

	t.Run("includes task loop", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "include-loop", "--file", "src/test/tasks/tasks.yaml")
		require.Error(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "task loop detected")
	})

	t.Run("run cmd-set-variable with --set", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "cmd-set-variable", "--set", "REPLACE_ME=replacedWith--setvar", "--set", "UNICORNS=defense", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "I'm set from a runner var - replacedWith--setvar")
		require.Contains(t, stdErr, "I'm set from a new --set var - defense")
	})

	t.Run("run remote-import", func(t *testing.T) {
		t.Parallel()

		// get current git revision
		gitRev, err := e2e.GetGitRevision()
		if err != nil {
			return
		}
		setVar := fmt.Sprintf("GIT_REVISION=%s", gitRev)
		stdOut, stdErr, err := e2e.Maru("run", "remote-import", "--set", setVar, "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "defenseunicorns is a pretty ok company")
	})

	t.Run("run rerun-tasks", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "rerun-tasks", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
	})

	t.Run("run rerun-tasks-child", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "rerun-tasks-child", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
	})

	t.Run("run rerun-tasks-recursive", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "rerun-tasks-recursive", "--file", "src/test/tasks/tasks.yaml")
		require.Error(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "task loop detected")
	})

	t.Run("test includes paths", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "foobar", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "echo foo")
		require.Contains(t, stdErr, "echo bar")
	})

	t.Run("test action with multiple include tasks", func(t *testing.T) {
		t.Parallel()
		gitRev, err := e2e.GetGitRevision()
		if err != nil {
			return
		}
		setVar := fmt.Sprintf("GIT_REVISION=%s", gitRev)

		stdOut, stdErr, err := e2e.Maru("run", "more-foobar", "--set", setVar, "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "echo foo")
		require.Contains(t, stdErr, "echo bar")
		require.Contains(t, stdErr, "defenseunicorns is a pretty ok company")
	})

	t.Run("test action with multiple nested include tasks", func(t *testing.T) {
		t.Parallel()
		gitRev, err := e2e.GetGitRevision()
		if err != nil {
			return
		}
		setVar := fmt.Sprintf("GIT_REVISION=%s", gitRev)

		stdOut, stdErr, err := e2e.Maru("run", "extra-foobar", "--set", setVar, "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "echo foo")
		require.Contains(t, stdErr, "echo bar")
		require.Contains(t, stdErr, "defenseunicorns")
		require.Contains(t, stdErr, "defenseunicorns is a pretty ok company")
	})

	t.Run("test variable passing to included tasks", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "more-foo", "--set", "FOO_VAR=success", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "success")
		require.Contains(t, stdErr, "foo")
		require.Contains(t, stdErr, "bar")
		require.NotContains(t, stdErr, "default")
	})

	t.Run("run list tasks", func(t *testing.T) {
		t.Parallel()
		gitRev, err := e2e.GetGitRevision()
		if err != nil {
			return
		}
		setVar := fmt.Sprintf("GIT_REVISION=%s", gitRev)

		stdOut, stdErr, err := e2e.Maru("run", "--list", setVar, "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "echo-env-var")
		require.Contains(t, stdErr, "Test that env vars take precedence")
		require.Contains(t, stdErr, "remote-import")
		require.Contains(t, stdErr, "action")
	})

	t.Run("run --list-all tasks", func(t *testing.T) {
		t.Parallel()
		gitRev, err := e2e.GetGitRevision()
		if err != nil {
			return
		}
		setVar := fmt.Sprintf("GIT_REVISION=%s", gitRev)

		stdOut, stdErr, err := e2e.Maru("run", "--list-all", "--set", setVar, "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "echo-env-var")
		require.Contains(t, stdErr, "Test that env vars take precedence")
		require.Contains(t, stdErr, "foo:foobar")
		require.Contains(t, stdErr, "remote:echo-var")
	})

	t.Run("test bad call to zarf tools wait-for", func(t *testing.T) {
		t.Parallel()
		_, stdErr, err := e2e.Maru("run", "wait-fail", "--file", "src/test/tasks/tasks.yaml")
		require.Error(t, err)
		require.Contains(t, stdErr, "Waiting for")
	})

	t.Run("test successful call to zarf tools wait-for (requires Zarf on path)", func(t *testing.T) {
		t.Parallel()
		_, stderr, err := e2e.Maru("run", "wait-success", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err)
		require.Contains(t, stderr, "succeeded")
	})

	t.Run("test task to load env vars using the envPath key", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "env-from-file", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, e2e.Arch)
		require.Contains(t, stdErr, "not-a-secret")
		require.Contains(t, stdErr, "3000")
		require.Contains(t, stdErr, "$env/**/*var with#special%chars!")
		require.Contains(t, stdErr, "env var from calling task - not-a-secret")
		require.Contains(t, stdErr, "overwritten env var - 8080")
	})

	t.Run("test that variables of type file and setting dir from a variable are processed correctly", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "file-and-dir", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "SECRET_KEY=not-a-secret")
	})

	t.Run("test that env vars get used for variables that do not have a default set", func(t *testing.T) {
		t.Parallel()
		os.Setenv("RUN_LOG_LEVEL", "debug")
		os.Setenv("RUN_TO_BE_OVERWRITTEN", "env-var")
		stdOut, stdErr, err := e2e.Maru("run", "echo-env-var", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.NotContains(t, stdErr, "default")
		require.Contains(t, stdErr, "env-var")
		require.Contains(t, stdErr, "DEBUG")
	})

	t.Run("test calling an included task directly", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "foo:foobar", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "echo foo")
		require.Contains(t, stdErr, "echo bar")
	})

	t.Run("test calling a remote included task directly", func(t *testing.T) {
		t.Parallel()
		gitRev, err := e2e.GetGitRevision()
		if err != nil {
			return
		}
		setVar := fmt.Sprintf("GIT_REVISION=%s", gitRev)
		stdOut, stdErr, err := e2e.Maru("run", "remote:echo-var", "--set", setVar, "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "defenseunicorns is a pretty ok company")
	})
}
