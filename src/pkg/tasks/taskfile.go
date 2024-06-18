package tasks

import (
	"github.com/defenseunicorns/maru-runner/src/pkg/variables"
	"github.com/defenseunicorns/maru-runner/src/types"
)

const (
	defaultTaskName = "default"
)

// TasksFile represents the contents of a tasks file
type TasksFile struct {
	Includes  []map[string]string                                          `json:"includes,omitempty" jsonschema:"description=List of local task files to include"`
	Env       map[string]string                                            `json:"env,omitempty" jsonschema:"description=Environment variables to set for all tasks"`
	Variables []variables.InteractiveVariable[variables.ExtraVariableInfo] `json:"variables,omitempty" jsonschema:"description=Definitions and default values for variables used in run.yaml"`
	Tasks     []*types.Task                                                `json:"tasks" jsonschema:"description=The list of tasks that can be run"`

	src      string
	dirPath  string
	filePath string
}
