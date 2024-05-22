// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package utils provides utility fns for maru
package utils

import (
	"regexp"
	"strings"
	"text/template"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/pkg/variables"
	"github.com/defenseunicorns/maru-runner/src/types"
	goyaml "github.com/goccy/go-yaml"
)

// TemplateTaskActionsWithInputs templates a task's actions with the given inputs
func TemplateTaskActionsWithInputs(task types.Task, withs map[string]string) ([]types.Action, error) {
	data := map[string]map[string]string{
		"inputs": {},
	}

	// get inputs from "with" map
	for name := range withs {
		data["inputs"][name] = withs[name]
	}

	// use default if not populated in data
	for name := range task.Inputs {
		if current, ok := data["inputs"][name]; !ok || current == "" {
			data["inputs"][name] = task.Inputs[name].Default
		}
	}

	b, err := goyaml.Marshal(task.Actions)
	if err != nil {
		return nil, err
	}

	t, err := template.New("template task actions").Option("missingkey=error").Delims("${{", "}}").Parse(string(b))
	if err != nil {
		return nil, err
	}

	var templated strings.Builder

	if err := t.Execute(&templated, data); err != nil {
		return nil, err
	}

	result := templated.String()

	var templatedActions []types.Action

	return templatedActions, goyaml.Unmarshal([]byte(result), &templatedActions)
}

// TemplateString replaces ${...} with the value from the template map
func TemplateString[T any](setVariableMap variables.SetVariableMap[T], s string) string {
	// Create a regular expression to match ${...}
	re := regexp.MustCompile(`\${(.*?)}`)

	// template string using values from the set variable map
	result := re.ReplaceAllStringFunc(s, func(matched string) string {
		varName := strings.TrimSuffix(strings.TrimPrefix(matched, "${"), "}")

		if value, ok := config.GetExtraEnv()[varName]; ok {
			return value
		}
		if value, ok := setVariableMap[varName]; ok {
			return value.Value
		}
		return matched // If the key is not found, keep the original substring
	})

	return result
}
