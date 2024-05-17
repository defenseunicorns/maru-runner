// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024-Present Defense Unicorns

package variables

import (
	"fmt"
	"regexp"
	"strings"
)

// SetVariableMap represents a map of variable names to their set values
type SetVariableMap map[string]*SetVariable

// GetSetVariable gets a variable set within a VariableConfig by its name
func (vc *VariableConfig) GetSetVariable(name string) (variable *SetVariable, ok bool) {
	variable, ok = vc.setVariableMap[name]
	return variable, ok
}

// GetSetVariables gets the variables set within a VariableConfig
func (vc *VariableConfig) GetSetVariables() SetVariableMap {
	return vc.setVariableMap
}

// PopulateVariables handles setting the active variables within a VariableConfig's SetVariableMap
func (vc *VariableConfig) PopulateVariables(variables []InteractiveVariable, presetVariables map[string]string) error {
	for name, value := range presetVariables {
		vc.SetVariable(name, value, false, false, "")
	}

	for _, variable := range variables {
		_, present := vc.setVariableMap[variable.Name]

		// Variable is present, no need to continue checking
		if present {
			vc.setVariableMap[variable.Name].Sensitive = variable.Sensitive
			vc.setVariableMap[variable.Name].AutoIndent = variable.AutoIndent
			vc.setVariableMap[variable.Name].Type = variable.Type
			if err := vc.CheckVariablePattern(variable.Name, variable.Pattern); err != nil {
				return err
			}
			continue
		}

		// First set default (may be overridden by prompt)
		vc.SetVariable(variable.Name, variable.Default, variable.Sensitive, variable.AutoIndent, variable.Type)

		// Variable is set to prompt the user
		if variable.Prompt {
			// Prompt the user for the variable
			val, err := vc.prompt(variable)

			if err != nil {
				return err
			}

			vc.SetVariable(variable.Name, val, variable.Sensitive, variable.AutoIndent, variable.Type)
		}

		if err := vc.CheckVariablePattern(variable.Name, variable.Pattern); err != nil {
			return err
		}
	}

	return nil
}

// SetVariable sets a variable in a VariableConfig's SetVariableMap
func (vc *VariableConfig) SetVariable(name, value string, sensitive bool, autoIndent bool, varType VariableType) {
	vc.setVariableMap[name] = &SetVariable{
		Variable: Variable{
			Name:       name,
			Sensitive:  sensitive,
			AutoIndent: autoIndent,
			Type:       varType,
		},
		Value: value,
	}
}

// CheckVariablePattern checks to see if a current variable is set to a value that matches its pattern
func (vc *VariableConfig) CheckVariablePattern(name, pattern string) error {
	if variable, ok := vc.setVariableMap[name]; ok {
		if regexp.MustCompile(pattern).MatchString(variable.Value) {
			return nil
		}

		return fmt.Errorf("provided value for variable %q does not match pattern %q", name, pattern)
	}

	return fmt.Errorf("variable %q was not found in the current variable map", name)
}

// GetAllTemplates gets all of the current templates stored in the VariableConfig
func (vc *VariableConfig) GetAllTemplates() map[string]*TextTemplate {
	templateMap := vc.applicationTemplates

	for key, variable := range vc.setVariableMap {
		// Variable keys are always uppercase in the format i.e. ###ZARF_VAR_KEY### or ###UDS_VAR_KEY###
		templateMap[strings.ToUpper(fmt.Sprintf("###%s_VAR_%s###", vc.templatePrefix, key))] = &TextTemplate{
			Value:      variable.Value,
			Sensitive:  variable.Sensitive,
			AutoIndent: variable.AutoIndent,
			Type:       variable.Type,
		}
	}

	for _, constant := range vc.constants {
		// Constant keys are always uppercase in the format i.e. ###ZARF_CONST_KEY###
		templateMap[strings.ToUpper(fmt.Sprintf("###%s_CONST_%s###", vc.templatePrefix, constant.Name))] = &TextTemplate{
			Value:      constant.Value,
			AutoIndent: constant.AutoIndent,
		}
	}

	return templateMap
}
