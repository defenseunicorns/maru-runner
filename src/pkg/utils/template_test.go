// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

package utils

import (
	"testing"

	zarfTypes "github.com/defenseunicorns/zarf/src/types"

	"github.com/defenseunicorns/maru-runner/src/types"
	zarfUtils "github.com/defenseunicorns/zarf/src/pkg/utils"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestTemplateString(t *testing.T) {
	type args struct {
		templateMap map[string]*zarfUtils.TextTemplate
		s           string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "single replacement",
			args: args{
				templateMap: map[string]*zarfUtils.TextTemplate{
					"${VAR1}": {Value: "replacement1"},
				},
				s: "This is a ${VAR1} string.",
			},
			want: "This is a replacement1 string.",
		},
		{
			name: "multiple replacements",
			args: args{
				templateMap: map[string]*zarfUtils.TextTemplate{
					"${VAR1}": {Value: "replacement1"},
					"${VAR2}": {Value: "replacement2"},
				},
				s: "This is a ${VAR1} and ${VAR2} string.",
			},
			want: "This is a replacement1 and replacement2 string.",
		},
		{
			name: "no replacements",
			args: args{
				templateMap: map[string]*zarfUtils.TextTemplate{},
				s:           "This string has no variables.",
			},
			want: "This string has no variables.",
		},
		{
			name: "missing replacement",
			args: args{
				templateMap: map[string]*zarfUtils.TextTemplate{
					"${VAR1}": {Value: "replacement1"},
				},
				s: "This is a ${VAR1} and ${MISSING_VAR} string.",
			},
			want: "This is a replacement1 and ${MISSING_VAR} string.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TemplateString(tt.args.templateMap, tt.args.s); got != tt.want {
				t.Errorf("TemplateString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateTaskActionsWithInputs(t *testing.T) {
	type args struct {
		task  types.Task
		withs map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    []types.Action
		wantErr bool
	}{
		{
			name: "successful template with inputs",
			args: args{
				task: types.Task{
					Name:        "test-task",
					Description: "test task",
					Inputs: map[string]types.InputParameter{
						"test-input": {Default: "default1", Description: "test input"},
					},
					Actions: []types.Action{
						{TaskReference: "test-task", With: map[string]string{"test-input": "value1"}},
					},
				},
				withs: map[string]string{"test-input": "value1"},
			},
			want: []types.Action{
				{TaskReference: "test-task", With: map[string]string{"test-input": "value1"}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TemplateTaskActionsWithInputs(tt.args.task, tt.args.withs)
			if (err != nil) != tt.wantErr {
				t.Errorf("TemplateTaskActionsWithInputs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreFields(types.Action{}, "ZarfComponentAction")); diff != "" {
				t.Errorf("TemplateTaskActionsWithInputs() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPopulateTemplateMap(t *testing.T) {
	type args struct {
		zarfVariables []zarfTypes.ZarfPackageVariable
		setVariables  map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]*zarfUtils.TextTemplate
	}{
		{
			name: "Populate with no overrides",
			args: args{
				zarfVariables: []zarfTypes.ZarfPackageVariable{
					{Name: "TEST_VAR", Default: "default_value", Sensitive: false, AutoIndent: false, Type: "string"},
				},
				setVariables: map[string]string{},
			},
			want: map[string]*zarfUtils.TextTemplate{
				"${TEST_VAR}": {
					Value:      "default_value",
					Sensitive:  false,
					AutoIndent: false,
					Type:       "string",
				},
			},
		},
		{
			name: "Populate with overrides",
			args: args{
				zarfVariables: []zarfTypes.ZarfPackageVariable{
					{Name: "TEST_VAR", Default: "default_value", Sensitive: false, AutoIndent: false, Type: "string"},
				},
				setVariables: map[string]string{
					"TEST_VAR": "overridden_value",
				},
			},
			want: map[string]*zarfUtils.TextTemplate{
				"${TEST_VAR}": {
					Value:      "overridden_value",
					Sensitive:  false,
					AutoIndent: false,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PopulateTemplateMap(tt.args.zarfVariables, tt.args.setVariables)
			for key, wantVal := range tt.want {
				if gotVal, ok := got[key]; ok {
					if diff := cmp.Diff(wantVal, gotVal); diff != "" {
						t.Errorf("PopulateTemplateMap() mismatch (-want +got):\n%s", diff)
					}
				} else {
					t.Errorf("PopulateTemplateMap() missing key: %v", key)
				}
			}
		})
	}
}
