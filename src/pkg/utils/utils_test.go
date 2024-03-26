// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

package utils

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMergeEnv(t *testing.T) {
	type args struct {
		env1 []string
		env2 []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Merge with no duplicates",
			args: args{
				env1: []string{"PATH=/usr/bin", "HOME=/home/user"},
				env2: []string{"GO111MODULE=on", "CGO_ENABLED=0"},
			},
			want: []string{"CGO_ENABLED=0", "PATH=/usr/bin", "HOME=/home/user", "GO111MODULE=on"},
		},
		{
			name: "Merge with duplicates",
			args: args{
				env1: []string{"PATH=/usr/bin", "HOME=/home/user"},
				env2: []string{"PATH=/usr/local/bin", "EDITOR=vim"},
			},
			want: []string{"PATH=/usr/bin", "HOME=/home/user", "EDITOR=vim"},
		},
		{
			name: "Merge with empty first env",
			args: args{
				env1: []string{},
				env2: []string{"PATH=/usr/local/bin", "EDITOR=vim"},
			},
			want: []string{"PATH=/usr/local/bin", "EDITOR=vim"},
		},
		{
			name: "Merge with empty second env",
			args: args{
				env1: []string{"PATH=/usr/bin", "HOME=/home/user"},
				env2: []string{},
			},
			want: []string{"PATH=/usr/bin", "HOME=/home/user"},
		},
		{
			name: "Merge both envs empty",
			args: args{
				env1: []string{},
				env2: []string{},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MergeEnv(tt.args.env1, tt.args.env2)
			if diff := cmp.Diff(tt.want, got, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
				t.Errorf("MergeEnv() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
func TestFormatEnvVar(t *testing.T) {
	type args struct {
		name  string
		value string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Format standard variable",
			args: args{
				name:  "PATH",
				value: "/usr/bin:/bin",
			},
			want: "INPUT_PATH=/usr/bin:/bin",
		},
		{
			name: "Format empty value",
			args: args{
				name:  "EMPTY_VAR",
				value: "",
			},
			want: "INPUT_EMPTY_VAR=",
		},
		{
			name: "Format variable with spaces",
			args: args{
				name:  "WITH_SPACES",
				value: "value with spaces",
			},
			want: "INPUT_WITH_SPACES=value with spaces",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatEnvVar(tt.args.name, tt.args.value); got != tt.want {
				t.Errorf("FormatEnvVar() = %v, want %v", got, tt.want)
			}
		})
	}
}
