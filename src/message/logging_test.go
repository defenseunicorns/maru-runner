// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

package message

import (
	"bytes"
	"log/slog"
	"slices"
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
			expected: "DEBUG test",
		},
		"InfoInfoLevel": {
			setLevel: InfoLevel,
			logLevel: InfoLevel,
			expected: "INFO test",
		},
		"InfoWarnLevel": {
			setLevel: InfoLevel,
			logLevel: WarnLevel,
			expected: "WARNING test",
		},
		"WarnInfoLevel": {
			setLevel: WarnLevel,
			logLevel: InfoLevel,
			expected: "",
		},
		"InfoErrorLevel": {
			setLevel: InfoLevel,
			logLevel: ErrorLevel,
			expected: "ERROR test",
		},
		"TraceInfoLevel": {
			setLevel: TraceLevel,
			logLevel: InfoLevel,
			expected: "INFO test",
		},
		"TraceDebugLevel": {
			setLevel: TraceLevel,
			logLevel: DebugLevel,
			expected: "DEBUG test",
		},
		"TraceErrorLevel": {
			setLevel: TraceLevel,
			logLevel: ErrorLevel,
			expected: "ERROR test",
		},
		"ErrorWarnLevel": {
			setLevel: ErrorLevel,
			logLevel: WarnLevel,
			expected: "",
		},
		"ErrorErrorLevel": {
			setLevel: ErrorLevel,
			logLevel: ErrorLevel,
			expected: "ERROR test",
		},
		"ErrorInfoLevel": {
			setLevel: ErrorLevel,
			logLevel: InfoLevel,
			expected: "",
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
			case ErrorLevel:
				maruLogger.Error("test")
			}
			content := outBuf.String()
			// remove color codes
			content = pterm.RemoveColorFromString(content)
			// remove extra whitespace from the output
			content = strings.TrimSpace(content)
			parts := strings.Split(tc.expected, " ")
			for _, part := range parts {
				if !strings.Contains(content, part) {
					t.Errorf("Expected debug message to contain '%s', but it didn't: (%s)", part, content)
				}
			}
			// if the set level is Trace and the log level is Debug, then we
			// expect extra	debug lines to be printed. Conversely, if it's trace
			// but not Debug, then we expect no extra debug lines to be printed.
			partsOutput := strings.Split(content, " ")
			// when debugging with TraceLevel, spliting on spaces will result in a slice
			// like so:
			// []string{
			//   "DEBUG",
			//   "",
			//   "",
			//   "2024-09-19T10:21:16-05:00",
			//   "",
			//   "-",
			//   "",
			//   "test\n└",
			//   "(/Users/clint/go/github.com/defenseunicorns/maru-runner/src/message/slog.go:56)",
			// }
			//
			// here we sort the slice to move the timestamp to the front,
			// then compact to remove them. The result should be a slice of
			// 6 eleements.
			//
			// While debugging without trace level, we expect the same slice
			// except there is no file name and line number, so it would have 5
			// elements.
			slices.Sort(partsOutput)
			partsOutput = slices.Compact(partsOutput)
			expectedLen := 3
			if tc.logLevel == DebugLevel {
				expectedLen = 5
			}
			if tc.setLevel == TraceLevel && tc.logLevel == DebugLevel {
				expectedLen = 6
			}

			if len(partsOutput) > expectedLen {
				t.Errorf("Expected debug message to contain timestamp, but it didn't: (%s)", content)
			}
		})
	}

}
