// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package message provides a rich set of functions for displaying messages to the user.
package message

import (
	"bytes"
	"log/slog"
	"strings"

	"testing"

	"github.com/pterm/pterm"
)

func Test_LogLevel_Diff(t *testing.T) {
	maruLogger := slog.New(MaruHandler{})

	cases := map[string]struct {
		// the level we're set to log at with SetLogLevel(). We expect logs with
		// a lower level to be ignored.
		setLevel LogLevel
		// the level which we will log, e.g. SLog.Debug(), SLog.Info(), etc.
		logLevel LogLevel
		// the expected output of the log. We special case DebugLevel as it
		// should contain a timestamp.
		expected string
	}{
		"DebugLevel": {
			setLevel: DebugLevel,
			logLevel: DebugLevel,
			expected: "DEBUG test", // spacing doesn't matter as we split this on whitespace
		},
		"InfoLevel": {
			setLevel: InfoLevel,
			logLevel: InfoLevel,
			expected: "INFO  test", // 2 spaces between INFO and <message>
		},
		"InfoWarnLevel": {
			setLevel: InfoLevel,
			logLevel: WarnLevel,
			expected: "WARNING  test",
		},
		"WarnInfoLevel": {
			setLevel: WarnLevel,
			logLevel: InfoLevel,
			expected: "",
		},
		"InfoTraceLevel": {
			setLevel: InfoLevel,
			logLevel: TraceLevel,
			expected: "ERROR   test", // Trace/errors have 3 spaces for some reason
		},
		"TraceInfoLevel": {
			setLevel: TraceLevel,
			logLevel: InfoLevel,
			expected: "",
		},
		"TraceLevel": {
			setLevel: TraceLevel,
			logLevel: TraceLevel,
			expected: "ERROR   test",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			SetLogLevel(tc.setLevel)

			// set the underlying writer, like we do in utils/utils.go
			var outBuf bytes.Buffer
			pterm.SetDefaultOutput(&outBuf)

			switch tc.logLevel {
			case DebugLevel:
				maruLogger.Debug("test")
			case InfoLevel:
				maruLogger.Info("test")
			case WarnLevel:
				maruLogger.Warn("test")
			case TraceLevel:
				maruLogger.Error("test")
			}
			content := outBuf.String()
			// remove color codes
			content = pterm.RemoveColorFromString(content)
			// remove extra whitespace from the output
			content = strings.TrimSpace(content)

			// if we're not debugging, just check the content directly
			if tc.logLevel != DebugLevel {
				if content != tc.expected {
					t.Errorf("Expected '%s', got %s", tc.expected, content)
				}
				return
			}

			// if we're debugging then we'd expect a timestamp so the exact
			// match should fail
			if content == tc.expected {
				t.Errorf("Expected timestamp to not match '%s', got %s", tc.expected, content)
			}
			parts := strings.Split(tc.expected, " ")
			for _, part := range parts {
				if !strings.Contains(content, part) {
					t.Errorf("Expected debug message to contain '%s', but it didn't", part)
				}
			}
		})
	}

}
