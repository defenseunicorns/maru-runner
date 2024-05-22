// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

package utils

import (
	"testing"

	"github.com/defenseunicorns/maru-runner/src/config"

	"github.com/defenseunicorns/maru-runner/src/pkg/variables"
)

func Test_TemplateString(t *testing.T) {

	type args struct {
		vars     variables.SetVariableMap[string]
		extraEnv map[string]string
		s        string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test precedence of extraEnv over setVariableMap",
			args: args{
				vars:     variables.SetVariableMap[string]{"ENV1": {Value: "fromSet1"}, "ENV2": {Value: "fromSet2"}, "ENV3": {Value: "fromSet3"}, "ENV4": {Value: "fromSet4"}},
				extraEnv: map[string]string{"ENV1": "fromExtra1", "ENV3": "fromExtra3", "ENV4": "fromExtra4"},
				s:        "${ENV1} ${ENV2} ${ENV3} ${ENV4}",
			},
			want: "fromExtra1 fromSet2 fromExtra3 fromExtra4",
		},
		{
			name: "Test with no setVariableMap",
			args: args{
				vars:     nil,
				extraEnv: map[string]string{"ENV1": "fromExtra1", "ENV3": "fromExtra3", "ENV4": "fromExtra4"},
				s:        "${ENV1} ${ENV2} ${ENV3} ${ENV4}",
			},
			want: "fromExtra1 ${ENV2} fromExtra3 fromExtra4",
		},
		{
			name: "Test with no extraEnv",
			args: args{
				vars:     variables.SetVariableMap[string]{"ENV1": {Value: "fromSet1"}, "ENV3": {Value: "fromSet3"}, "ENV4": {Value: "fromSet4"}},
				extraEnv: nil,
				s:        "${ENV1} ${ENV2} ${ENV3} ${ENV4}",
			},
			want: "fromSet1 ${ENV2} fromSet3 fromSet4",
		},
		{
			name: "Test no set or extraEnv",
			args: args{
				vars:     nil,
				extraEnv: nil,
				s:        "${ENV1} ${ENV2} ${ENV3} ${ENV4}",
			},
			want: "${ENV1} ${ENV2} ${ENV3} ${ENV4}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.ClearExtraEnv()
			for k, v := range tt.args.extraEnv {
				config.AddExtraEnv(k, v)
			}

			if got := TemplateString(tt.args.vars, tt.args.s); got != tt.want {
				t.Errorf("TemplateString() [%s] got = %q, want %q", tt.name, got, tt.want)
			}
		})
	}

}
