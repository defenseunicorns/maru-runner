// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package message contains functions to print messages to the screen
package message

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pterm/pterm"
)

// LogLevel is the level of logging to display.
type LogLevel int

const (
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel LogLevel = iota
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
	// TraceLevel level. Designates finer-grained informational events than the Debug.
	TraceLevel
)

// logLevel is the log level for the runner
var logLevel = InfoLevel

// logFile acts as a buffer for logFile generation
var logFile *os.File

// UseLogFile writes output to stderr and a logFile.
func UseLogFile(dir string) (io.Writer, error) {
	// Prepend the log filename with a timestamp.
	ts := time.Now().Format("2006-01-02-15-04-05")

	var err error
	logFile, err = os.CreateTemp(dir, fmt.Sprintf("maru-%s-*.log", ts))
	if err != nil {
		return nil, err
	}

	return logFile, nil
}

// LogFileLocation returns the location of the log file.
func LogFileLocation() string {
	if logFile == nil {
		return ""
	}
	return logFile.Name()
}

// SetLogLevel sets the log level.
func SetLogLevel(lvl LogLevel) {
	logLevel = lvl
	if logLevel >= DebugLevel {
		pterm.EnableDebugMessages()
	}
}

// GetLogLevel returns the current log level.
func GetLogLevel() LogLevel {
	return logLevel
}
