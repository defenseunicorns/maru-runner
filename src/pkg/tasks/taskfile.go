package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "github.com/goccy/go-yaml"

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

	dirPath  string
	filePath string
	taskMap  map[string]*Task
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

func Parse(filePath string) (*TasksFile, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	tasks := &TasksFile{
		filePath: filepath.Base(filePath),
		dirPath:  filepath.Dir(filePath),
		taskMap:  make(map[string]*Task),
	}

	err = yaml.Unmarshal(file, tasks)
	if err != nil {
		return nil, err
	}

	for _, t := range tasks.Tasks {
		if _, ok := tasks.taskMap[t.Name]; ok {
			return nil, fmt.Errorf(`found duplicate task definition for "%s"`, t.Name)
		}

		if strings.Contains(t.Name, ":") {
			return nil, fmt.Errorf(`invalid task name "%s" (use of ":" is reserved for included tasks)`, t.Name)
		}

		tasks.taskMap[t.Name] = t
	}

	return tasks, nil
}

func (tf *TasksFile) Resolve(taskName string) (*TaskRunner, error) {
	if taskName == "" {
		taskName = defaultTaskName
	}

	if t, ok := tf.taskMap[taskName]; ok {
		return NewRunner(t, tf), nil
	}

	return nil, fmt.Errorf(`Task "%s" not found`, taskName)
}
