// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021-Present The Zarf Authors

// Package message provides a rich set of functions for displaying messages to the user.
package message

import (
	"context"
	"log/slog"
)

var (
	// SLogHandler sets the default structured log handler for messages
	SLogHandler = slog.New(MaruHandler{})
)

// MaruHandler is a simple handler that implements the slog.Handler interface
type MaruHandler struct{}

// Enabled is always set to true as Maru logging functions are already aware of if they are allowed to be called
func (z MaruHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// WithAttrs is not suppported
func (z MaruHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return z
}

// WithGroup is not supported
func (z MaruHandler) WithGroup(_ string) slog.Handler {
	return z
}

// Handle prints the respective logging function in zarf
// This function ignores any key pairs passed through the record
func (z MaruHandler) Handle(_ context.Context, record slog.Record) error {
	level := record.Level
	message := record.Message

	switch level {
	case slog.LevelDebug:
		debugf("%s", message)
	case slog.LevelInfo:
		infof("%s", message)
	case slog.LevelWarn:
		warnf("%s", message)
	case slog.LevelError:
		errorf("%s", message)
	}
	return nil
}
