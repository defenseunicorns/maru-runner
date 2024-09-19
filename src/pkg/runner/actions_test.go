// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

package runner

import (
	"reflect"
	"slices"
	"testing"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/types"

	"github.com/defenseunicorns/maru-runner/src/pkg/variables"

	"github.com/stretchr/testify/require"
)

func Test_getUniqueTaskActions(t *testing.T) {
	t.Parallel()
	type args struct {
		actions []types.Action
	}
	tests := []struct {
		name string
		args args
		want []types.Action
	}{
		{
			name: "No duplicates",
			args: args{
				actions: []types.Action{
					{TaskReference: "task1"},
					{TaskReference: "task2"},
				},
			},
			want: []types.Action{
				{TaskReference: "task1"},
				{TaskReference: "task2"},
			},
		},
		{
			name: "With duplicates",
			args: args{
				actions: []types.Action{
					{TaskReference: "task1"},
					{TaskReference: "task1"},
					{TaskReference: "task2"},
				},
			},
			want: []types.Action{
				{TaskReference: "task1"},
				{TaskReference: "task2"},
			},
		},
		{
			name: "All duplicates",
			args: args{
				actions: []types.Action{
					{TaskReference: "task1"},
					{TaskReference: "task1"},
					{TaskReference: "task1"},
				},
			},
			want: []types.Action{
				{TaskReference: "task1"},
			},
		},
		{
			name: "Empty slice",
			args: args{
				actions: nil,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := getUniqueTaskActions(tt.args.actions); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getUniqueTaskActions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertWaitToCmd(t *testing.T) {
	type args struct {
		wait    types.ActionWait
		timeout *int
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Cluster wait command",
			args: args{
				wait: types.ActionWait{
					Cluster: &types.ActionWaitCluster{
						Kind:       "pod",
						Identifier: "my-pod",
						Condition:  "Ready",
						Namespace:  "default",
					},
				},
				timeout: IntPtr(300),
			},
			want:    "zarf tools wait-for pod my-pod Ready -n default --timeout 300s",
			wantErr: false,
		},
		{
			name: "Network wait command",
			args: args{
				wait: types.ActionWait{
					Network: &types.ActionWaitNetwork{
						Protocol: "http",
						Address:  "http://example.com",
						Code:     200,
					},
				},
				timeout: IntPtr(60),
			},
			want:    "zarf tools wait-for http http://example.com 200 --timeout 60s",
			wantErr: false,
		},
		{
			name: "Invalid wait action",
			args: args{
				wait:    types.ActionWait{},
				timeout: IntPtr(30),
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertWaitToCmd(tt.args.wait, tt.args.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertWaitToCmd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("convertWaitToCmd() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func IntPtr(i int) *int {
	return &i
}

func Test_validateActionableTaskCall(t *testing.T) {
	type args struct {
		inputTaskName string
		inputs        map[string]types.InputParameter
		withs         map[string]string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Valid task call with all required inputs",
			args: args{
				inputTaskName: "testTask",
				inputs: map[string]types.InputParameter{
					"input1": {Required: true, Default: ""},
					"input2": {Required: true, Default: ""},
				},
				withs: map[string]string{
					"input1": "value1",
					"input2": "value2",
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid task call with missing required input",
			args: args{
				inputTaskName: "testTask",
				inputs: map[string]types.InputParameter{
					"input1": {Required: true, Default: ""},
					"input2": {Required: true, Default: ""},
				},
				withs: map[string]string{
					"input1": "value1",
				},
			},
			wantErr: true,
		},
		{
			name: "Valid task call with default value for missing input",
			args: args{
				inputTaskName: "testTask",

				inputs: map[string]types.InputParameter{
					"input1": {Required: true, Default: "defaultValue"},
					"input2": {Required: true, Default: ""},
				},
				withs: map[string]string{
					"input2": "value2",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateActionableTaskCall(tt.args.inputTaskName, tt.args.inputs, tt.args.withs); (err != nil) != tt.wantErr {
				t.Errorf("validateActionableTaskCall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunner_performAction(t *testing.T) {
	type fields struct {
		TasksFile      types.TasksFile
		TaskNameMap    map[string]bool
		envFilePath    string
		variableConfig *variables.VariableConfig[variables.ExtraVariableInfo]
	}
	type args struct {
		action types.Action
		inputs map[string]types.InputParameter
		withs  map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add more test cases
		// https://github.com/defenseunicorns/maru-runner/issues/143
		{
			name: "failed action processing due to invalid command",
			fields: fields{
				TasksFile:      types.TasksFile{},
				TaskNameMap:    make(map[string]bool),
				envFilePath:    "",
				variableConfig: GetMaruVariableConfig(),
			},
			args: args{
				action: types.Action{
					TaskReference: "",
					With: map[string]string{
						"cmd": "exit 1",
					},
					BaseAction: &types.BaseAction[variables.ExtraVariableInfo]{
						Description: "Test action for failure scenario",
						Wait:        nil,
					},
				},
			},
		},
		{
			name: "Unable to open path",
			fields: fields{
				TasksFile:      types.TasksFile{},
				TaskNameMap:    make(map[string]bool),
				envFilePath:    "test/path",
				variableConfig: GetMaruVariableConfig(),
			},
			args: args{
				action: types.Action{
					TaskReference: "",
					With: map[string]string{
						"cmd": "zarf tools wait-for pod my-pod Running",
					},
					BaseAction: &types.BaseAction[variables.ExtraVariableInfo]{
						Description: "Test action for wait command",
						Wait: &types.ActionWait{
							Cluster: &types.ActionWaitCluster{
								Kind:       "pod",
								Identifier: "my-pod",
								Condition:  "Running",
							},
						},
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{
				TasksFile:      tt.fields.TasksFile,
				TaskNameMap:    tt.fields.TaskNameMap,
				envFilePath:    tt.fields.envFilePath,
				variableConfig: tt.fields.variableConfig,
			}
			err := r.performAction(tt.args.action, tt.args.withs, tt.args.inputs)
			if (err != nil) != tt.wantErr {
				t.Errorf("performAction() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunner_processAction(t *testing.T) {
	type fields struct {
		TasksFile      types.TasksFile
		TaskNameMap    map[string]bool
		envFilePath    string
		variableConfig *variables.VariableConfig[variables.ExtraVariableInfo]
	}
	type args struct {
		task   types.Task
		action types.Action
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "successful action processing",
			fields: fields{
				TasksFile:      types.TasksFile{},
				TaskNameMap:    map[string]bool{},
				envFilePath:    "",
				variableConfig: GetMaruVariableConfig(),
			},
			args: args{
				task: types.Task{
					Name: "testTask",
				},
				action: types.Action{
					TaskReference: "testTaskRef",
				},
			},
			want: true,
		},
		{
			name: "action processing with same task and action reference",
			fields: fields{
				TasksFile:      types.TasksFile{},
				TaskNameMap:    map[string]bool{},
				envFilePath:    "",
				variableConfig: GetMaruVariableConfig(),
			},
			args: args{
				task: types.Task{
					Name: "testTask",
				},
				action: types.Action{
					TaskReference: "testTask",
				},
			},
			want: false,
		},
		{
			name: "action processing with empty task reference",
			fields: fields{
				TasksFile:      types.TasksFile{},
				TaskNameMap:    map[string]bool{},
				envFilePath:    "",
				variableConfig: GetMaruVariableConfig(),
			},
			args: args{
				task: types.Task{
					Name: "testTask",
				},
				action: types.Action{
					TaskReference: "",
				},
			},
			want: false,
		},
		{
			name: "action processing with non-empty task reference and different task and action reference names",
			fields: fields{
				TasksFile:      types.TasksFile{},
				TaskNameMap:    map[string]bool{},
				envFilePath:    "",
				variableConfig: GetMaruVariableConfig(),
			},
			args: args{
				task: types.Task{
					Name: "testTask",
				},
				action: types.Action{
					TaskReference: "differentTaskRef",
				},
			},
			want: true,
		},
		{
			name: "action processing with task reference already processed",
			fields: fields{
				TasksFile: types.TasksFile{
					Tasks: []types.Task{
						{
							Name: "testTaskRef:subTask",
						},
					},
				},
				TaskNameMap:    map[string]bool{},
				envFilePath:    "",
				variableConfig: GetMaruVariableConfig(),
			},
			args: args{
				task: types.Task{
					Name: "testTask",
				},
				action: types.Action{
					TaskReference: "testTaskRef",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{
				TasksFile:      tt.fields.TasksFile,
				TaskNameMap:    tt.fields.TaskNameMap,
				envFilePath:    tt.fields.envFilePath,
				variableConfig: tt.fields.variableConfig,
			}
			if got := r.processAction(tt.args.task, tt.args.action); got != tt.want {
				t.Errorf("processAction() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunner_GetBaseActionCfg(t *testing.T) {
	type args struct {
		cfg      types.ActionDefaults
		a        types.BaseAction[string]
		vars     variables.SetVariableMap[string]
		extraEnv map[string]string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "ActionDefaults used when no overrides",
			args: args{
				cfg: types.ActionDefaults{
					Env: []string{"ENV1=fromDefault", "ENV2=xyz"},
				},
				a: types.BaseAction[string]{},
			},
			want: []string{"ENV1=fromDefault", "ENV2=xyz"},
		},
		{
			name: "extraEnv overrides defaults",
			args: args{
				cfg: types.ActionDefaults{
					Env: []string{"ENV1=fromDefault", "ENV2=xyz1"},
				},
				a:        types.BaseAction[string]{},
				vars:     variables.SetVariableMap[string]{"ENV1": {Value: "fromSet"}},
				extraEnv: map[string]string{"ENV1": "fromExtra"},
			},
			want: []string{"ENV1=fromDefault", "ENV2=xyz1", "ENV1=fromSet", "ENV1=fromExtra"},
		},
		{
			name: "extraEnv adds to defaults",
			args: args{
				cfg: types.ActionDefaults{
					Env: []string{"ENV1=fromDefault", "ENV2=xyz1"},
				},
				a:        types.BaseAction[string]{},
				vars:     variables.SetVariableMap[string]{"ENV1": {Value: "fromSet"}},
				extraEnv: map[string]string{"ENV3": "fromExtra"},
			},
			want: []string{"ENV1=fromDefault", "ENV2=xyz1", "ENV1=fromSet", "ENV3=fromExtra"},
		},
		{
			name: "extraEnv adds and overrides defaults",
			args: args{
				cfg: types.ActionDefaults{
					Env: []string{"ENV1=fromDefault", "ENV2=xyz1"},
				},
				a:        types.BaseAction[string]{},
				vars:     variables.SetVariableMap[string]{"ENV4": {Value: "fromSet"}},
				extraEnv: map[string]string{"ENV2": "alsoFromEnv", "ENV3": "fromExtra"},
			},
			want: []string{"ENV1=fromDefault", "ENV2=xyz1", "ENV4=fromSet", "ENV2=alsoFromEnv", "ENV3=fromExtra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.ClearExtraEnv()
			for k, v := range tt.args.extraEnv {
				config.AddExtraEnv(k, v)
			}

			got := GetBaseActionCfg(tt.args.cfg, tt.args.a, tt.args.vars)
			slices.Sort(got.Env)
			slices.Sort(tt.want)
			require.Equal(t, tt.want, got.Env, "The returned Env array did not match what was wanted")
		})
	}

}
