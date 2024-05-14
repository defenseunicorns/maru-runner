// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

// Package utils provides utility fns for maru
package utils

import (
	"fmt"
	"maps"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/defenseunicorns/maru-runner/src/types"
	goyaml "github.com/goccy/go-yaml"

	"github.com/defenseunicorns/maru-runner/src/config"
	zarfTypes "github.com/defenseunicorns/zarf/src/types"
)

// TextTemplate represents a value to be templated into a text file.
// todo: should be grabbing from Zarf but it's now private: https://github.com/defenseunicorns/zarf/issues/2395
type TextTemplate struct {
	Sensitive  bool
	AutoIndent bool
	Type       zarfTypes.VariableType
	Value      string
}

// PopulateTemplateMap creates a template variable map
func PopulateTemplateMap(zarfVariables []zarfTypes.ZarfPackageVariable, setVariables map[string]string) map[string]*TextTemplate {
	// populate text template (ie. Zarf var) with the following precedence: default < env var < set var
	templateMap := make(map[string]*TextTemplate)
	for _, variable := range zarfVariables {
		templatedVariableName := fmt.Sprintf("${%s}", variable.Name)
		textTemplate := &TextTemplate{
			Sensitive:  variable.Sensitive,
			AutoIndent: variable.AutoIndent,
			Type:       variable.Type,
		}

		if v := os.Getenv(fmt.Sprintf("%s_%s", strings.ToUpper(config.EnvPrefix), variable.Name)); v != "" {
			textTemplate.Value = v
		} else {
			textTemplate.Value = variable.Default
		}
		templateMap[templatedVariableName] = textTemplate
	}

	setVariablesTemplateMap := make(map[string]*TextTemplate)
	for name, value := range setVariables {
		setVariablesTemplateMap[fmt.Sprintf("${%s}", name)] = &TextTemplate{
			Value: value,
		}
	}

	maps.Copy(templateMap, setVariablesTemplateMap)
	return templateMap
}

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
func TemplateString(templateMap map[string]*TextTemplate, s string) string {
	// Create a regular expression to match ${...}
	re := regexp.MustCompile(`\${(.*?)}`)

	// template string using values from the template map
	result := re.ReplaceAllStringFunc(s, func(matched string) string {
		if value, ok := templateMap[matched]; ok {
			return value.Value
		}
		return matched // If the key is not found, keep the original substring
	})
	return result
}
