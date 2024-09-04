// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package utils provides utility fns for maru
package utils

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/message"
	"github.com/defenseunicorns/maru-runner/src/pkg/variables"
	"github.com/defenseunicorns/maru-runner/src/types"
	goyaml "github.com/goccy/go-yaml"
)

// Update this to handle vars and inputs and only operate on singular action
// TemplateTaskActionsWithInputs templates a task's actions with the given inputs
// TemplateTaskActionsWithInputs templates a task's actions with the given inputs
// TemplateTaskActionsWithInputs templates a task's actions with the given inputs
func TemplateTaskActionsWithInputs[T any](inputs map[string]types.InputParameter, action types.Action, withs map[string]string, vms variables.SetVariableMap[T]) (types.Action, error) {
	message.SLog.Debug(fmt.Sprintf("Entering TemplateTaskActionsWithInputs for %s", action.Cmd))
	data := map[string]map[string]string{
		"inputs":    {},
		"variables": {},
	}

	// get inputs from "with" map
	for name := range withs {
		data["inputs"][name] = withs[name]
	}

	// get vars from "vms" map
	for name := range vms {
		data["variables"][name] = vms[name].Value
		//message.SLog.Debug(fmt.Sprintf("Current var in iteration %s", data["variables"][name]))
	}

	// use default if not populated in data
	for name := range inputs {
		if current, ok := data["inputs"][name]; !ok || current == "" {
			data["inputs"][name] = inputs[name].Default
		}
	}

	b, err := goyaml.Marshal(action)
	if err != nil {
		return action, err
	}

	t, err := template.New("template task actions").Option("missingkey=error").Delims("${{", "}}").Parse(string(b))
	if err != nil {
		return action, err
	}

	var templated strings.Builder

	if err := t.Execute(&templated, data); err != nil {
		return action, err
	}

	result := templated.String()

	var templatedActions types.Action
	if err := goyaml.Unmarshal([]byte(result), &templatedActions); err != nil {
		return action, err
	}

	// Pretty print the YAML
	// prettyPrintedYAML, err := goyaml.Marshal(templatedActions)
	// if err != nil {
	// 	return action, err
	// }

	// fmt.Println("Pretty Printed YAML:\n", string(prettyPrintedYAML))

	return templatedActions, nil
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

// // TemplateAndEvalActionConditional evaluates a condition using Go templates
// func TemplateAndEvalActionConditional(condition string, context map[string]interface{}) (bool, error) {
// 	if condition == "" {
// 		return true, nil
// 	}
// 	// Parse the condition as a Go template
// 	tmpl, err := template.New("template task action conditional").Option("missingkey=error").Delims("${{", "}}").Parse(condition)
// 	if err != nil {
// 		return false, fmt.Errorf("failed to parse condition: %w", err)
// 	}

// 	// Execute the template with the provided context
// 	var buf bytes.Buffer
// 	err = tmpl.Execute(&buf, context)
// 	if err != nil {
// 		return false, fmt.Errorf("failed to execute condition: %w", err)
// 	}

// 	// Evaluate the result of the template execution
// 	result := buf.String()
// 	return result == "true", nil
// }

// TemplateAndEvalActionConditional evaluates a condition using Go templates
func TemplateAndEvalActionConditional(condition string, context map[string]interface{}) (bool, error) {
	if condition == "" {
		return true, nil
	}
	// Parse the condition as a Go template
	tmpl, err := template.New("template task action conditional").Option("missingkey=error").Delims("${{", "}}").Parse(condition)
	if err != nil {
		return false, fmt.Errorf("failed to parse condition: %w", err)
	}

	// Execute the template with the provided context
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, context)
	if err != nil {
		return false, fmt.Errorf("failed to execute condition: %w", err)
	}

	// Evaluate the result of the template execution
	result := buf.String()
	return result == "true", nil
}
