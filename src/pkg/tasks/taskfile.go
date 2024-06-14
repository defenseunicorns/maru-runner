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
	Tasks     []*Task                                                      `json:"tasks" jsonschema:"description=The list of tasks that can be run"`

	src      string
	dirPath  string
	filePath string
}

// Task represents a single task
type Task struct {
	Name        string                    `json:"name" jsonschema:"description=Name of the task"`
	Description string                    `json:"description,omitempty" jsonschema:"description=Description of the task"`
	Actions     []types.Action            `json:"actions,omitempty" jsonschema:"description=Actions to take when running the task"`
	Steps       []types.Step              `json:"steps,omitempty" jsonschema:"description=Actions to take when running the task"`
	Inputs      map[string]InputParameter `json:"inputs,omitempty" jsonschema:"description=Input parameters for the task"`
	Outputs     map[string]string         `json:"outputs,omitempty" jsonschema:"description=Outputs from the task"`
}

// InputParameter represents a single input parameter for a task, to be used w/ `with`
type InputParameter struct {
	Description       string `json:"description" jsonschema:"description=Description of the parameter,required"`
	DeprecatedMessage string `json:"deprecatedMessage,omitempty" jsonschema:"description=Message to display when the parameter is deprecated"`
	Required          bool   `json:"required,omitempty" jsonschema:"description=Whether the parameter is required,default=true"`
	Default           string `json:"default,omitempty" jsonschema:"description=Default value for the parameter"`
}
