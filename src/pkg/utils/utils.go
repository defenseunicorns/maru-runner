// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

// Package utils provides utility fns for maru
package utils

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/defenseunicorns/zarf/src/pkg/message"
	"github.com/pterm/pterm"
)

// UseLogFile writes output to stderr and a logFile.
func UseLogFile() {
	// LogWriter is the stream to write logs to.
	var LogWriter io.Writer

	// Write logs to stderr and a buffer for logFile generation.
	var logFile *os.File

	// Prepend the log filename with a timestamp.
	ts := time.Now().Format("2006-01-02-15-04-05")

	var err error
	if logFile != nil {
		// Use the existing log file if logFile is set
		LogWriter = io.MultiWriter(os.Stderr, logFile)
		pterm.SetDefaultOutput(LogWriter)
	} else {
		// Try to create a temp log file if one hasn't been made already
		if logFile, err = os.CreateTemp("", fmt.Sprintf("runner-%s-*.log", ts)); err != nil {
			message.WarnErr(err, "Error saving a log file to a temporary directory")
		} else {
			LogWriter = io.MultiWriter(os.Stderr, logFile)
			pterm.SetDefaultOutput(LogWriter)
			msg := fmt.Sprintf("Saving log file to %s", logFile.Name())
			message.Note(msg)
		}
	}
}

// MergeEnv merges two environment variable arrays,
// replacing variables found in env2 with variables from env1
// otherwise appending the variable from env1 to env2
func MergeEnv(env1, env2 []string) []string {
	for _, s1 := range env1 {
		replaced := false
		for j, s2 := range env2 {
			if strings.Split(s1, "=")[0] == strings.Split(s2, "=")[0] {
				env2[j] = s1
				replaced = true
			}
		}
		if !replaced {
			env2 = append(env2, s1)
		}
	}
	return env2
}

// FormatEnvVar format environment variables replacing non-alphanumeric characters with underscores and adding INPUT_ prefix
func FormatEnvVar(name, value string) string {
	// replace all non-alphanumeric characters with underscores
	name = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(name, "_")
	name = strings.ToUpper(name)
	// prefix with INPUT_ (same as GitHub Actions)
	return fmt.Sprintf("INPUT_%s=%s", name, value)
}
