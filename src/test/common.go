// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package test contains e2e tests for the runner
package test

import (
	"context"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/defenseunicorns/pkg/exec"
	"github.com/defenseunicorns/pkg/helpers"
	"github.com/stretchr/testify/require"
)

// MaruE2ETest Struct holding common fields most of the tests will utilize.
type MaruE2ETest struct {
	MaruBinPath       string
	Arch              string
	ApplianceMode     bool
	ApplianceModeKeep bool
	RunClusterTests   bool
	CommandLog        []string
}

// GetCLIName looks at the OS and CPU architecture to determine which Maru binary needs to be run.
func GetCLIName() string {
	var binaryName string
	if runtime.GOOS == "linux" {
		binaryName = "maru"
	} else if runtime.GOOS == "darwin" {
		if runtime.GOARCH == "arm64" {
			binaryName = "maru-mac-apple"
		} else {
			binaryName = "maru-mac-intel"
		}
	}
	return binaryName
}

var logRegex = regexp.MustCompile(`Saving log file to (?P<logFile>.*?\.log)`)

// Maru executes a run command.
func (e2e *MaruE2ETest) Maru(args ...string) (string, string, error) {
	e2e.CommandLog = append(e2e.CommandLog, strings.Join(args, " "))
	return exec.CmdWithContext(context.TODO(), exec.Config{Print: true}, e2e.MaruBinPath, args...)
}

// CleanFiles removes files and directories that have been created during the test.
func (e2e *MaruE2ETest) CleanFiles(files ...string) {
	for _, file := range files {
		_ = os.RemoveAll(file)
	}
}

// GetMismatchedArch determines what architecture our tests are running on,
// and returns the opposite architecture.
func (e2e *MaruE2ETest) GetMismatchedArch() string {
	switch e2e.Arch {
	case "arm64":
		return "amd64"
	default:
		return "arm64"
	}
}

// GetLogFileContents gets the log file contents from a given run's std error.
func (e2e *MaruE2ETest) GetLogFileContents(t *testing.T, stdErr string) string {
	get, err := helpers.MatchRegex(logRegex, stdErr)
	require.NoError(t, err)
	logFile := get("logFile")
	logContents, err := os.ReadFile(logFile)
	require.NoError(t, err)
	return string(logContents)
}

// GetMaruVersion returns the current build version
func (e2e *MaruE2ETest) GetMaruVersion(t *testing.T) string {
	// Get the version of the CLI
	stdOut, stdErr, err := e2e.Maru("version")
	require.NoError(t, err, stdOut, stdErr)
	return strings.Trim(stdOut, "\n")
}

// GetGitRevision returns the current git revision
func (e2e *MaruE2ETest) GetGitRevision() (string, error) {
	out, _, err := exec.Cmd(exec.Config{Print: true}, "git", "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

// HelmDepUpdate runs 'helm dependency update .' on the given path
func (e2e *MaruE2ETest) HelmDepUpdate(t *testing.T, path string) {
	cmd := "helm"
	args := strings.Split("dependency update .", " ")
	cfg := exec.Config{Print: true, Dir: path}
	_, _, err := exec.CmdWithContext(context.TODO(), cfg, cmd, args...)
	require.NoError(t, err)
}
