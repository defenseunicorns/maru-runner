// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package message provides a rich set of functions for displaying messages to the user.
package message

import (
	"fmt"
	"os"

	"github.com/defenseunicorns/pkg/helpers"
	"github.com/pterm/pterm"
)

const padding = "    "

// Public interfaces for Maru message settings
var (
	// NoProgress sets whether the default spinners and progress bars should use fancy animations
	NoProgress bool
)

// ProgressBar is a struct used to drive a pterm ProgressbarPrinter.
type ProgressBar struct {
	progress *pterm.ProgressbarPrinter
}

// NewProgressBar creates a new ProgressBar instance from a total value and a format.
var NewProgressBar = func(total int64, format string, a ...any) helpers.ProgressWriter {
	var progress *pterm.ProgressbarPrinter
	if NoProgress {
		infof(format, a...)
	} else {
		title := fmt.Sprintf(format, a...)
		progress, _ = pterm.DefaultProgressbar.
			WithTotal(int(total)).
			WithShowCount(false).
			WithTitle(padding + title).
			WithRemoveWhenDone(true).
			WithMaxWidth(termWidth).
			WithWriter(os.Stderr).
			Start()
	}

	return &ProgressBar{
		progress: progress,
	}
}

// Write updates the ProgressBar with the number of bytes in a buffer as the completed progress.
func (p *ProgressBar) Write(data []byte) (int, error) {
	n := len(data)
	if p.progress != nil {
		if p.progress.Current+n >= p.progress.Total {
			// @RAZZLE TODO: This is a hack to prevent the progress bar from going over 100% and causing TUI ugliness.
			overflow := p.progress.Current + n - p.progress.Total
			p.progress.Total += overflow + 1
		}
		p.progress.Add(n)
	}
	return n, nil
}

// Close stops the ProgressBar from continuing.
func (p *ProgressBar) Close() error {
	if p.progress != nil {
		_, err := p.progress.Stop()
		return err
	}
	return nil
}

// Updatef updates the ProgressBar with new text.
func (p *ProgressBar) Updatef(format string, a ...any) {
	text := fmt.Sprintf(format, a...)
	if NoProgress {
		debugPrinter(2, text)
		return
	}
	p.progress.UpdateTitle(padding + text)
}

// Successf marks the ProgressBar as successful in the CLI.
func (p *ProgressBar) Successf(format string, a ...any) {
	p.Close()
	successf(format, a...)
}

// Failf marks the ProgressBar as failed in the CLI.
func (p *ProgressBar) Failf(format string, a ...any) {
	p.Close()
	errorf(format, a...)
}
