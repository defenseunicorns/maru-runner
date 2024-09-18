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

// func Test_LogLevel_Basic(t *testing.T) {
// 	sLog := slog.New(MaruHandler{})
// 	cases := map[string]struct {
// 		level    LogLevel
// 		expected string
// 	}{
// 		"DebugLevel": {
// 			level:    DebugLevel,
// 			expected: "DEBUG test",
// 		},
// 		"InfoLevel": {
// 			level:    InfoLevel,
// 			expected: "INFO  test",
// 		},
// 		"WarnLevel": {
// 			level:    WarnLevel,
// 			expected: "WARNING  test",
// 		},
// 		"TraceLevel": {
// 			level:    TraceLevel,
// 			expected: "ERROR   test", // 3 spaces for some reason
// 		},
// 	}

// 	for name, tc := range cases {
// 		t.Run(name, func(t *testing.T) {
// 			SetLogLevel(tc.level)
// 			var outBuf bytes.Buffer
// 			// set the underlying writer, like we do in utils/utils.go
// 			pterm.SetDefaultOutput(&outBuf)

// 			switch tc.level {
// 			case DebugLevel:
// 				sLog.Debug("test")
// 			case InfoLevel:
// 				sLog.Info("test")
// 			case WarnLevel:
// 				sLog.Warn("test")
// 			case TraceLevel:
// 				sLog.Error("test")
// 			}
// 			content := outBuf.String()
// 			// outBuf.Reset()
// 			content = pterm.RemoveColorFromString(content)
// 			content = strings.TrimSpace(content)
// 			if tc.level != DebugLevel {
// 				if content != tc.expected {
// 					t.Errorf("Expected '%s', got %s", tc.expected, content)
// 				}
// 			} else {
// 				parts := strings.Split(tc.expected, " ")
// 				for _, part := range parts {
// 					if !strings.Contains(content, part) {
// 						t.Errorf("Expected debug message to contain '%s', but it didn't", part)
// 					}
// 				}
// 			}
// 		})
// 	}

// }

func Test_LogLevel_Diff(t *testing.T) {
	// SetLogLevel(DebugLevel)
	// if GetLogLevel() != DebugLevel {
	// 	t.Errorf("Expected DebugLevel, got %d", GetLogLevel())
	// }

	sLog := slog.New(MaruHandler{})

	// enabled := sLog.Enabled(context.Background(), slog.LevelDebug)
	// dbl := slog.Level(DebugLevel)
	// dbl := slog.Level(WarnLevel)

	// var outBuf bytes.Buffer
	// // set the underlying writer, like we do in utils/utils.go
	// // SetLogLevel(InfoLevel)
	// SetLogLevel(WarnLevel)
	// pterm.SetDefaultOutput(&outBuf)
	// // sLog.Info("test")
	// sLog.Warn("test")
	// content := outBuf.String()
	// outBuf.Reset()
	// content = pterm.RemoveColorFromString(content)
	// content = strings.TrimSpace(content)

	// if content != "INFO  test" {
	// 	t.Errorf("Expected 'INFO  test', got %s", content)
	// }
	// enabled := sLog.Enabled(context.Background(), slog.Level(DebugLevel))
	// enabled := sLog.Enabled(context.Background(), slog.Level(WarnLevel))
	// if !enabled {
	// 	t.Errorf("Expected enabled to be true, got %t", enabled)
	// }

	cases := map[string]struct {
		// the level we're set to log at with SetLogLevel()
		setLevel LogLevel
		// the level which we will log, e.g. SLog.Debug(), SLog.Info(), etc.
		logLevel LogLevel
		// the expected output of the log. We special case DebugLevel as it
		// contains output with a timestamp
		expected string
	}{
		"DebugLevel": {
			setLevel: DebugLevel,
			logLevel: DebugLevel,
			expected: "DEBUG test",
		},
		"InfoLevel": {
			setLevel: InfoLevel,
			logLevel: InfoLevel,
			expected: "INFO  test",
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
			expected: "ERROR   test", // 3 spaces for some reason
		},
		"TraceInfoLevel": {
			setLevel: TraceLevel,
			logLevel: InfoLevel,
			expected: "",
		},
		"TraceLevel": {
			setLevel: TraceLevel,
			logLevel: TraceLevel,
			expected: "ERROR   test", // 3 spaces for some reason
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			SetLogLevel(tc.setLevel)
			var outBuf bytes.Buffer
			// set the underlying writer, like we do in utils/utils.go
			pterm.SetDefaultOutput(&outBuf)

			switch tc.logLevel {
			case DebugLevel:
				sLog.Debug("test")
			case InfoLevel:
				sLog.Info("test")
			case WarnLevel:
				sLog.Warn("test")
			case TraceLevel:
				sLog.Error("test")
			}
			content := outBuf.String()
			// outBuf.Reset()
			content = pterm.RemoveColorFromString(content)
			content = strings.TrimSpace(content)
			if tc.logLevel != DebugLevel {
				if content != tc.expected {
					t.Errorf("Expected '%s', got %s", tc.expected, content)
				}
			} else {
				parts := strings.Split(tc.expected, " ")
				for _, part := range parts {
					if !strings.Contains(content, part) {
						t.Errorf("Expected debug message to contain '%s', but it didn't", part)
					}
				}
			}
		})
	}

}
