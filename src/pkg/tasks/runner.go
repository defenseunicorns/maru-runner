package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/defenseunicorns/maru-runner/src/types"

	"github.com/risor-io/risor"
	"github.com/risor-io/risor/object"
	ros "github.com/risor-io/risor/os"
	"github.com/risor-io/risor/os/localfs"
)

type TaskRunner struct {
	task      *Task
	tasksFile *TasksFile

	inputs      map[string]string
	stepOutputs map[string]interface{}
}

func NewRunner(task *Task, tasksFile *TasksFile) *TaskRunner {
	runner := &TaskRunner{
		task:        task,
		tasksFile:   tasksFile,
		inputs:      make(map[string]string),
		stepOutputs: make(map[string]interface{}),
	}

	for k, input := range task.Inputs {
		runner.inputs[k] = input.Default
	}

	if task.Actions != nil {
		task.Steps = make([]types.Step, 0)

		for _, a := range task.Actions {
			task.Steps = append(task.Steps, ToStep(a))
		}
	}

	return runner
}

func (r *TaskRunner) Run(ctx context.Context) error {
	ctx, err := r.getContext(ctx)
	fmt.Printf("starting task '%s': %v\n", r.task.Name, r.inputs)

	if err != nil {
		return err
	}

	for _, step := range r.task.Steps {
		fmt.Printf("starting step '%s.%s'\n", r.task.Name, step.ID)

		// child task
		if step.Uses != "" {
			child, err := r.tasksFile.Resolve(step.Uses)
			if err != nil {
				return err
			}

			child.SetInputs(step.With)
			if err = child.Run(ctx); err != nil {
				return err
			}

			r.setStepOutput(step.ID, child.Outputs(ctx))
			continue
		}

		// if step.WorkDir != "" {
		// 	vm.Chdir(step.WorkDir)
		// }

		var result object.Object
		var err error

		if step.Script != "" {
			result, err = r.eval(ctx, step.Script)
		} else if step.Cmd != "" {
			// shell, shellArgs := exec.GetOSShell(*step.Shell)
			result, err = r.exec(ctx, "sh", []string{"-e", "-c", step.Cmd})
		} else {
			continue
		}

		if err != nil {
			return err
		}

		r.setStepOutput(step.ID, result)
	}

	fmt.Printf("finished running task '%s': %v\n", r.task.Name, r.Outputs(ctx))

	return nil
}

func (r *TaskRunner) SetInputs(inputs map[string]string) error {
	for k, v := range inputs {
		if _, ok := r.inputs[k]; !ok {
			return fmt.Errorf("'%s' is not a valid input for task '%s'", k, r.task.Name)
		}

		r.inputs[k] = v
	}

	// TODO: check if required fields are missing

	return nil
}

func (r *TaskRunner) Outputs(ctx context.Context) object.Object {
	var code strings.Builder

	code.WriteString("m := {}")

	for k, value := range r.task.Outputs {
		if strings.HasPrefix(value, "${{") {
			expr := strings.TrimPrefix(value, "${{")
			expr = strings.TrimSuffix(expr, "}}")

			fmt.Fprintf(&code, `
m["%s"] = %s
`, k, expr)
		}
	}

	code.WriteString("\nm")

	out, err := r.eval(ctx, code.String())
	if err != nil {
		fmt.Println(err)
	}
	// out, _ := fromRisor(result)

	return out
}

func (r *TaskRunner) setStepOutput(stepID string, obj object.Object) error {
	if stepID != "" {
		value, err := fromRisor(obj)
		if err != nil {
			return err
		}

		r.stepOutputs[stepID] = value
	}

	return nil
}

func (r *TaskRunner) eval(ctx context.Context, expression string) (object.Object, error) {
	return risor.Eval(
		ctx,
		expression,
		risor.WithGlobal("inputs", r.inputs),
		risor.WithGlobal("steps", r.stepOutputs),
	)
}

func (r *TaskRunner) exec(ctx context.Context, shell string, args []string) (object.Object, error) {
	return risor.Eval(
		ctx,
		"exec(shell, args).stdout",
		risor.WithGlobal("shell", shell),
		risor.WithGlobal("args", args),
	)
}

func (r *TaskRunner) getContext(ctx context.Context) (context.Context, error) {
	if _, ok := ros.GetOS(ctx); ok {
		// virtual OS already set by parent runner
		return ctx, nil
	}

	workdir, err := localfs.New(ctx, localfs.WithBase(r.tasksFile.dirPath))

	if err != nil {
		return nil, err
	}

	return ros.WithOS(ctx, ros.NewVirtualOS(ctx,
		ros.WithMounts(map[string]*ros.Mount{
			"/workdir": {
				Source: workdir,
				Target: "/workdir",
			},
		}),
		ros.WithCwd("/workdir"),
		ros.WithStdout(os.Stdout),
		ros.WithStdin(os.Stdin),
	)), nil
}

func fromRisor(value object.Object) (interface{}, error) {
	switch obj := value.(type) {
	case *object.Int:
	case *object.Float:
	case *object.String:
		return obj.Interface(), nil
	case *object.Map:
		out := make(map[string]interface{})
		str, err := obj.MarshalJSON()
		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(str), &out)
		return out, nil
	case *object.List:
		out := make([]interface{}, obj.Size())
		str, err := obj.MarshalJSON()
		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(str), &out)
		return out, nil
		// json, err := obj.MarshalJSON()

		// if err != nil {
		// 	return "", fmt.Errorf("could not serialize output")
		// }

		// return string(json), nil
	case *object.NilType:
		return nil, nil
	}

	return "", fmt.Errorf("unsupported output type: %T", value)
}
