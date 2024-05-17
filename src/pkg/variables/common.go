// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024-Present Defense Unicorns

// Package variables contains functions for interacting with variables
package variables

import (
	"log/slog"
)

// VariableConfig represents a value to be templated into a text file.
type VariableConfig struct {
	templatePrefix string
	deprecatedKeys map[string]string

	setVariableMap SetVariableMap

	prompt func(variable InteractiveVariable) (value string, err error)
	logger *slog.Logger
}

// New creates a new VariableConfig
func New(templatePrefix string, deprecatedKeys map[string]string, prompt func(variable InteractiveVariable) (value string, err error), logger *slog.Logger) *VariableConfig {
	return &VariableConfig{
		templatePrefix: templatePrefix,
		deprecatedKeys: deprecatedKeys,
		setVariableMap: make(SetVariableMap),
		prompt:         prompt,
		logger:         logger,
	}
}
