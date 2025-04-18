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
		require.Contains(t, stdErr, "task looping exceeded max configured task stack")
	})

	t.Run("run direct loop", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "direct-loop", "--file", "src/test/tasks/loop-task.yaml")
		require.Error(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "task looping exceeded max configured task stack")
	})

	t.Run("includes intentional task loop", func(t *testing.T) {
		t.Parallel()

		// get current git revision
		gitRev, err := e2e.GetGitRevision()
		if err != nil {
			return
		}
		setVar := fmt.Sprintf("GIT_REVISION=%s", gitRev)
		stdOut, stdErr, err := e2e.Maru("run", "include-loop", "--set", setVar, "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "9")
		require.Contains(t, stdErr, "0")
	})

	t.Run("run cmd-set-variable with --set", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "cmd-set-variable", "--set", "REPLACE_ME=replacedWith--setvar", "--set", "UNICORNS=defense", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "I'm set from a runner var - replacedWith--setvar")
		require.Contains(t, stdErr, "I'm set from a new --set var - defense")
	})

	t.Run("run cmd-set-variable automatic MARU variable is false if overridden in cmd", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "cmd-set-variable", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "I was changed on the command line - MARU=hello")
	})

	t.Run("run cmd-set-variable automatic MARU variable is true", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "cmd-set-variable", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "I'm set automatically - MARU=true")
	})

	t.Run("run cmd-set-variable automatic MARU variable is true even if set", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "cmd-set-variable", "--file", "src/test/tasks/tasks.yaml", "--set", "MARU=false")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "I'm set automatically - MARU=true")
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

	t.Run("run remote-import back to local", func(t *testing.T) {
		t.Parallel()

		// get current git revision
		gitRev, err := e2e.GetGitRevision()
		if err != nil {
			return
		}
		setVar := fmt.Sprintf("GIT_REVISION=%s", gitRev)
		stdOut, stdErr, err := e2e.Maru("run", "remote-import-to-local", "--set", setVar, "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "baz")
	})

	t.Run("run remote-import gitlab", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "remote-gitlab:hello", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "hello")
		require.Contains(t, stdErr, "kitteh")
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
		require.Contains(t, stdErr, "task looping exceeded max configured task stack")
	})

	t.Run("run interactive (with --no-progress)", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "interactive", "--file", "src/test/tasks/tasks.yaml", "--no-progress")
		require.NoError(t, err, stdOut, stdErr)
		// Ensure there are no extra chars that will interrupt interactive programs (i.e. a spinner) when --no-progress is set
		require.Contains(t, stdErr, "\033[G ⬒ Spinning...\033[G ⬔ Spinning...\033[G ◨ Spinning...")
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

	t.Run("run --list tasks", func(t *testing.T) {
		t.Parallel()
		gitRev, err := e2e.GetGitRevision()
		if err != nil {
			return
		}
		setVar := fmt.Sprintf("GIT_REVISION=%s", gitRev)

		stdOut, stdErr, err := e2e.Maru("run", "--list", setVar, "--file", "src/test/tasks/tasks.yaml")

		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdOut, "echo-env-var")
		require.Contains(t, stdOut, "Test that env vars take precedence")
		require.Contains(t, stdOut, "remote-import")
		require.Contains(t, stdOut, "action")
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
		require.Contains(t, stdOut, "echo-env-var")
		require.Contains(t, stdOut, "Test that env vars take precedence")
		require.Contains(t, stdOut, "foo:foobar")
		require.Contains(t, stdOut, "remote:echo-var")
		require.Contains(t, stdOut, "remote-api:non-default")
	})

	t.Run("run --list=md tasks", func(t *testing.T) {
		t.Parallel()
		gitRev, err := e2e.GetGitRevision()
		if err != nil {
			return
		}
		setVar := fmt.Sprintf("GIT_REVISION=%s", gitRev)

		stdOut, stdErr, err := e2e.Maru("run", "--list=md", setVar, "--file", "src/test/tasks/tasks.yaml")

		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdOut, "|------|-------------|")
		require.Contains(t, stdOut, "echo-env-var")
		require.Contains(t, stdOut, "Test that env vars take precedence")
		require.Contains(t, stdOut, "remote-import")
		require.Contains(t, stdOut, "action")
	})

	t.Run("run --list-all=md tasks", func(t *testing.T) {
		t.Parallel()
		gitRev, err := e2e.GetGitRevision()
		if err != nil {
			return
		}
		setVar := fmt.Sprintf("GIT_REVISION=%s", gitRev)

		stdOut, stdErr, err := e2e.Maru("run", "--list-all=md", "--set", setVar, "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdOut, "|------|-------------|")
		require.Contains(t, stdOut, "echo-env-var")
		require.Contains(t, stdOut, "Test that env vars take precedence")
		require.Contains(t, stdOut, "foo:foobar")
		require.Contains(t, stdOut, "remote:echo-var")
	})

	t.Run("test bad call to zarf tools wait-for", func(t *testing.T) {
		t.Parallel()
		_, stdErr, err := e2e.Maru("run", "wait-fail", "--file", "src/test/tasks/tasks.yaml")
		require.Error(t, err)
		require.Contains(t, stdErr, "timed out after 1 seconds")
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
		require.Contains(t, stdErr, "not-a-secret")
		require.Contains(t, stdErr, "3000")
		require.Contains(t, stdErr, "$env/**/*var with#special%chars!")
		require.Contains(t, stdErr, "env var from calling task - not-a-secret")
		require.Contains(t, stdErr, "overwritten env var - 8080")
	})

	t.Run("test that setting dir from a variable is processed correctly", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "file-and-dir", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "SECRET_KEY=not-a-secret")
	})

	t.Run("test that setting an env var from a variable is processed correctly", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "env-templating", "--file", "src/test/tasks/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "hello-replaced")
	})

	t.Run("test that env vars get used for variables that do not have a default set", func(t *testing.T) {
		t.Parallel()
		os.Setenv("MARU_LOG_LEVEL", "debug")
		os.Setenv("MARU_TO_BE_OVERWRITTEN", "env-var")
		stdOut, stdErr, err := e2e.Maru("run", "echo-env-var", "--file", "src/test/tasks/tasks.yaml")
		os.Unsetenv("MARU_LOG_LEVEL")
		os.Unsetenv("MARU_TO_BE_OVERWRITTEN")
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

	// Conditional Tests
	t.Run("test calling a task with false conditional cmd comparing variables", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "false-conditional-with-var-cmd", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Skipping action false-conditional-with-var-cmd")
	})

	t.Run("test calling a task with true conditional cmd comparing variables", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "true-conditional-with-var-cmd", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "This should run because .variables.BAR = default-value")
	})

	t.Run("test calling a task with cmd no conditional", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "empty-conditional-cmd", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "This should run because there is no condition")
	})

	t.Run("test calling a task with false conditional comparing variables", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "false-conditional-task", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Skipping action included-task")
	})

	t.Run("test calling a task with true conditional comparing variables", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "true-conditional-task", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Task called successfully")
	})

	t.Run("test calling a task with no conditional comparing variables", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "empty-conditional-task", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Task called successfully")
	})

	t.Run("test calling a task with nested true conditional comparing variables and inputs", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "true-conditional-nested-task-comp-var-inputs", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "input val equals 5 and variable VAL1 equals 5")
	})
	t.Run("test calling a task with nested false conditional comparing variables and inputs", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "false-conditional-nested-task-comp-var-inputs", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Skipping action included-task-with-inputs")
	})

	t.Run("test calling a task with nested task true conditional comparing variables and inputs", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "true-conditional-nested-nested-task-comp-var-inputs", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Task called successfully")
	})
	t.Run("test calling a task with nested task false conditional comparing variables and inputs", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "false-conditional-nested-nested-task-comp-var-inputs", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Skipping action included-task")
	})

	t.Run("test calling a task with nested task calling a task with true conditional comparing variables and inputs", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "true-conditional-nested-nested-nested-task-comp-var-inputs", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Completed \"echo \\\"input val2 equals 5 and variable VAL1 equals 5\\\"\"")
	})
	t.Run("test calling a task with nested task calling a task with false conditional comparing variables and inputs", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "false-conditional-nested-nested-nested-task-comp-var-inputs", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Skipping action \"echo \\\"input val2 equals 7 and variable VAL1 equals 5\\\"\"")
	})

	t.Run("test calling a task with nested task calling a task with old style var as input true conditional comparing variables and inputs", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "true-condition-var-as-input-original-syntax-nested-nested-with-comp", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Completed \"echo \\\"input val2 equals 5 and variable VAL1 equals 5\\\"\"")
	})

	t.Run("test calling a task with nested task calling a task with new style var as input true conditional comparing variables and inputs", func(t *testing.T) {
		t.Parallel()
		stdOut, stdErr, err := e2e.Maru("run", "true-condition-var-as-input-new-syntax-nested-nested-with-comp", "--file", "src/test/tasks/conditionals/tasks.yaml")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Completed \"echo \\\"input val2 equals 5 and variable VAL1 equals 5\\\"\"")
	})

	t.Run("run successful pattern", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "--file", "src/test/tasks/more-tasks/pattern.yaml", "--set", "HELLO=HELLO")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "HELLO")
	})

	t.Run("run unsuccessful pattern", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "--file", "src/test/tasks/more-tasks/pattern.yaml", "--set", "HELLO=HI")
		require.Error(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "\"HELLO\" does not match pattern \"^HELLO$\"")
	})

	t.Run("dry run", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "--dry-run", "--file", "src/test/tasks/tasks.yaml", "env-from-file")
		require.NoError(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "Dry-running \"echo $MARU_ARCH\"")
		require.Contains(t, stdOut, "echo env var from calling task - $SECRET_KEY")
	})

	t.Run("redefined include", func(t *testing.T) {
		t.Parallel()

		stdOut, stdErr, err := e2e.Maru("run", "--file", "src/test/tasks/redefined-include.yaml")
		require.Error(t, err, stdOut, stdErr)
		require.Contains(t, stdErr, "task include \"foo\" attempted to be redefined")
	})
}
