// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

// Package utils provides utility fns for maru
package utils

import (
	"fmt"
	"io"
	"os"
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
