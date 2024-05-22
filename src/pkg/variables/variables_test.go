// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2024-Present Defense Unicorns

package variables

import (
	"errors"
	"reflect"
	"testing"
)

type testVariableInfo struct {
	Sensitive  bool
	AutoIndent bool
	Type       VariableType
}

func TestPopulateVariables(t *testing.T) {

	nonZeroTestVariableInfo := testVariableInfo{Sensitive: true, AutoIndent: true, Type: FileVariableType}

	type test struct {
		vc       VariableConfig[testVariableInfo]
		vars     []InteractiveVariable[testVariableInfo]
		presets  map[string]string
		wantErr  error
		wantVars SetVariableMap[testVariableInfo]
	}

	prompt := func(_ InteractiveVariable[testVariableInfo]) (value string, err error) { return "Prompt", nil }

	tests := []test{
		{
			vc: VariableConfig[testVariableInfo]{setVariableMap: SetVariableMap[testVariableInfo]{}},
			vars: []InteractiveVariable[testVariableInfo]{
				createInteractiveVariable("NAME", "", "", false, testVariableInfo{}),
			},
			presets: map[string]string{},
			wantErr: nil,
			wantVars: SetVariableMap[testVariableInfo]{
				"NAME": createSetVariable("NAME", "", "", testVariableInfo{})},
		},
		{
			vc: VariableConfig[testVariableInfo]{setVariableMap: SetVariableMap[testVariableInfo]{}},
			vars: []InteractiveVariable[testVariableInfo]{
				createInteractiveVariable("NAME", "", "Default", false, testVariableInfo{}),
			},
			presets: map[string]string{},
			wantErr: nil,
			wantVars: SetVariableMap[testVariableInfo]{
				"NAME": createSetVariable("NAME", "Default", "", testVariableInfo{}),
			},
		},
		{
			vc: VariableConfig[testVariableInfo]{setVariableMap: SetVariableMap[testVariableInfo]{}},
			vars: []InteractiveVariable[testVariableInfo]{
				createInteractiveVariable("NAME", "", "Default", false, testVariableInfo{}),
			},
			presets: map[string]string{"NAME": "Set"},
			wantErr: nil,
			wantVars: SetVariableMap[testVariableInfo]{
				"NAME": createSetVariable("NAME", "Set", "", testVariableInfo{}),
			},
		},
		{
			vc: VariableConfig[testVariableInfo]{setVariableMap: SetVariableMap[testVariableInfo]{}},
			vars: []InteractiveVariable[testVariableInfo]{
				createInteractiveVariable("NAME", "", "", false, nonZeroTestVariableInfo),
			},
			presets: map[string]string{},
			wantErr: nil,
			wantVars: SetVariableMap[testVariableInfo]{
				"NAME": createSetVariable("NAME", "", "", nonZeroTestVariableInfo),
			},
		},
		{
			vc: VariableConfig[testVariableInfo]{setVariableMap: SetVariableMap[testVariableInfo]{}},
			vars: []InteractiveVariable[testVariableInfo]{
				createInteractiveVariable("NAME", "", "", false, nonZeroTestVariableInfo),
			},
			presets: map[string]string{"NAME": "Set"},
			wantErr: nil,
			wantVars: SetVariableMap[testVariableInfo]{
				"NAME": createSetVariable("NAME", "Set", "", nonZeroTestVariableInfo),
			},
		},
		{
			vc: VariableConfig[testVariableInfo]{setVariableMap: SetVariableMap[testVariableInfo]{}, prompt: prompt},
			vars: []InteractiveVariable[testVariableInfo]{
				createInteractiveVariable("NAME", "", "", true, testVariableInfo{}),
			},
			presets: map[string]string{},
			wantErr: nil,
			wantVars: SetVariableMap[testVariableInfo]{
				"NAME": createSetVariable("NAME", "Prompt", "", testVariableInfo{}),
			},
		},
		{
			vc: VariableConfig[testVariableInfo]{setVariableMap: SetVariableMap[testVariableInfo]{}, prompt: prompt},
			vars: []InteractiveVariable[testVariableInfo]{
				createInteractiveVariable("NAME", "", "Default", true, testVariableInfo{}),
			},
			presets: map[string]string{},
			wantErr: nil,
			wantVars: SetVariableMap[testVariableInfo]{
				"NAME": createSetVariable("NAME", "Prompt", "", testVariableInfo{}),
			},
		},
		{
			vc: VariableConfig[testVariableInfo]{setVariableMap: SetVariableMap[testVariableInfo]{}, prompt: prompt},
			vars: []InteractiveVariable[testVariableInfo]{
				createInteractiveVariable("NAME", "", "", true, testVariableInfo{}),
			},
			presets: map[string]string{"NAME": "Set"},
			wantErr: nil,
			wantVars: SetVariableMap[testVariableInfo]{
				"NAME": createSetVariable("NAME", "Set", "", testVariableInfo{}),
			},
		},
	}

	for _, tc := range tests {
		gotErr := tc.vc.PopulateVariables(tc.vars, tc.presets)
		if gotErr != nil && tc.wantErr != nil {
			if gotErr.Error() != tc.wantErr.Error() {
				t.Fatalf("wanted err: %s, got err: %s", tc.wantErr, gotErr)
			}
		} else if gotErr != nil {
			t.Fatalf("got unexpected err: %s", gotErr)
		}

		gotVars := tc.vc.GetSetVariables()

		if len(gotVars) != len(tc.wantVars) {
			t.Fatalf("wanted vars len: %d, got vars len: %d", len(tc.wantVars), len(gotVars))
		}

		for key := range gotVars {
			if !reflect.DeepEqual(gotVars[key], tc.wantVars[key]) {
				t.Fatalf("for key %s: wanted var: %v, got var: %v", key, tc.wantVars[key], gotVars[key])
			}
		}
	}
}

func TestCheckVariablePattern(t *testing.T) {
	type test struct {
		vc   VariableConfig[testVariableInfo]
		name string
		want error
	}

	tests := []test{
		{
			vc:   VariableConfig[testVariableInfo]{setVariableMap: SetVariableMap[testVariableInfo]{}},
			name: "NAME",
			want: errors.New("variable \"NAME\" was not found in the current variable map"),
		},
		{
			vc: VariableConfig[testVariableInfo]{
				setVariableMap: SetVariableMap[testVariableInfo]{
					"NAME": createSetVariable("NAME", "name", "n[^a]me", testVariableInfo{}),
				},
			},
			name: "NAME",
			want: errors.New("provided value for variable \"NAME\" does not match pattern \"n[^a]me\""),
		},
		{
			vc: VariableConfig[testVariableInfo]{
				setVariableMap: SetVariableMap[testVariableInfo]{
					"NAME": createSetVariable("NAME", "name", "n[a-z]me", testVariableInfo{}),
				},
			},
			name: "NAME",
			want: nil,
		},
	}

	for _, tc := range tests {
		got := tc.vc.CheckVariablePattern(tc.name)
		if got != nil && tc.want != nil {
			if got.Error() != tc.want.Error() {
				t.Fatalf("wanted err: %s, got err: %s", tc.want, got)
			}
		} else if got != nil {
			t.Fatalf("got unexpected err: %s", got)
		}
	}
}

func createSetVariable(name, value, pattern string, extra testVariableInfo) *SetVariable[testVariableInfo] {
	return &SetVariable[testVariableInfo]{
		Value:    value,
		Variable: createVariable(name, pattern, extra),
	}
}

func createInteractiveVariable(name, pattern, def string, prompt bool, extra testVariableInfo) InteractiveVariable[testVariableInfo] {
	return InteractiveVariable[testVariableInfo]{
		Prompt:   prompt,
		Default:  def,
		Variable: createVariable(name, pattern, extra),
	}
}

func createVariable(name, pattern string, extra testVariableInfo) Variable[testVariableInfo] {
	return Variable[testVariableInfo]{
		Name:    name,
		Pattern: pattern,
		Extra:   extra,
	}
}
