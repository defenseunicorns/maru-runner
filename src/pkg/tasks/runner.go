package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/defenseunicorns/maru-runner/src/pkg/utils"
	"github.com/defenseunicorns/maru-runner/src/types"
	"github.com/defenseunicorns/pkg/helpers"

	yaml "github.com/goccy/go-yaml"
	"github.com/risor-io/risor"
	"github.com/risor-io/risor/object"
	ros "github.com/risor-io/risor/os"
	"github.com/risor-io/risor/os/localfs"
)

type TaskRunner struct {
	ctx       context.Context
	workDir   string
	taskFiles map[string]*TasksFile
	taskMap   map[string]*types.Task
}

type TaskRun struct {
	ctx         context.Context
	task        *types.Task
	steps       []*types.Step
	inputs      map[string]string
	env         map[string]string
	stepOutputs map[string]any
}

func NewRunner() *TaskRunner {
	runner := &TaskRunner{
		ctx:       context.Background(),
		taskFiles: make(map[string]*TasksFile),
		taskMap:   make(map[string]*types.Task),
	}

	return runner
}

func (r *TaskRunner) LoadRoot(src string) error {
	// Get the pwd
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error getting working directory: %s", err)
	}

	r.workDir = pwd

	return r.Load("", src)
}

func (r *TaskRunner) Load(key, src string) error {
	tasksFilePath := src

	if !filepath.IsAbs(tasksFilePath) {
		tasksFilePath = filepath.Join(r.workDir, tasksFilePath)
	}

	if _, loaded := r.taskFiles[tasksFilePath]; loaded {
		// tasksfile already loaded
		return nil
	}

	tasks := &TasksFile{
		src:      tasksFilePath,
		filePath: tasksFilePath, // will be different from `src` for remote files
		dirPath:  filepath.Dir(tasksFilePath),
	}

	// client := &getter.Client{
	// 	Src:  src,
	// 	Dst:  filepath.Join(pwd, ".maru", dst, "tasks.yaml"),
	// 	Pwd:  pwd,
	// 	Mode: getter.ClientModeFile,
	// }

	// fmt.Printf("loading '%s' into '%s'\n", client.Src, client.Dst)

	// if err := client.Get(); err != nil {
	// 	return nil, err
	// }

	fmt.Println("loading tasks from:", tasks.filePath)

	file, err := os.ReadFile(tasks.filePath)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(file, tasks)
	if err != nil {
		return err
	}

	r.taskFiles[tasks.src] = tasks

	for _, t := range tasks.Tasks {
		t.Name = getTaskName(key, t.Name)
		if _, ok := r.taskMap[t.Name]; ok {
			return fmt.Errorf("found duplicate task definition for '%s'", t.Name)
		}

		// if strings.Contains(taskName, ":") {
		// 	return fmt.Errorf("invalid task name '%s' (use of ':' is reserved for included tasks)", taskName)
		// }

		r.taskMap[t.Name] = t
	}

	for _, include := range tasks.Includes {
		for includeKey, src := range include {
			src = filepath.Join(tasks.dirPath, src)
			if err = r.Load(getTaskName(key, includeKey), src); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *TaskRunner) GetTasks() []string {
	taskNames := make([]string, 0)

	for key := range r.taskMap {
		taskNames = append(taskNames, key)
	}

	return taskNames
}

func (r *TaskRunner) Resolve(taskName string) (*TaskRun, error) {
	if taskName == "" {
		taskName = defaultTaskName
	}

	task, ok := r.taskMap[taskName]

	if !ok {
		return nil, fmt.Errorf("task '%s' is not defined", taskName)
	}

	ctx, err := r.getContext()
	if err != nil {
		return nil, err
	}

	return NewRun(task, ctx), nil
}

func (r *TaskRunner) Run(run *TaskRun) error {
	fmt.Printf("starting task '%s': %v\n", run.task.Name, run.inputs)

	for i, step := range run.steps {
		id := step.ID

		if id == "" {
			id = strconv.Itoa(i)
		}

		fmt.Printf("starting step '%s.%s'\n", run.task.Name, id)

		run.SetEnv(step.Env)

		// child task
		if step.Uses != "" {
			child, err := r.Resolve(step.Uses)
			if err != nil {
				return err
			}

			child.SetInputs(step.With)
			if err = r.Run(child); err != nil {
				return err
			}

			run.setStepOutput(step.ID, child.Outputs())
			continue
		}

		// if step.WorkDir != "" {
		// 	vm.Chdir(step.WorkDir)
		// }

		var result object.Object
		var err error

		if step.Script != "" {
			result, err = run.eval(step.Script)
		} else if step.Cmd != "" {
			// shell, shellArgs := exec.GetOSShell(*step.Shell)
			result, err = run.exec("sh", []string{"-e", "-c", step.Cmd})
		} else {
			continue
		}

		if err != nil {
			return err
		}

		run.setStepOutput(step.ID, result)
	}

	fmt.Printf("finished running task '%s': %v\n", run.task.Name, run.Outputs())

	return nil
}

func NewRun(task *types.Task, ctx context.Context) *TaskRun {
	run := &TaskRun{
		ctx:         ctx,
		task:        task,
		steps:       make([]*types.Step, 0),
		inputs:      make(map[string]string),
		env:         make(map[string]string),
		stepOutputs: make(map[string]any),
	}

	for k, input := range task.Inputs {
		run.inputs[k] = input.Default
	}

	if task.Actions != nil {
		for _, a := range task.Actions {
			run.steps = append(run.steps, ToStep(a))
		}
	} else {
		for _, s := range task.Steps {
			run.steps = append(run.steps, &s)
		}
	}

	return run
}

func (tr *TaskRun) SetEnv(env map[string]string) {
	maps.Copy(tr.env, env)
}

func (tr *TaskRun) SetInputs(inputs map[string]string) error {
	for k, v := range inputs {
		if _, ok := tr.task.Inputs[k]; !ok {
			return fmt.Errorf("'%s' is not a valid input for task '%s'", k, tr.task.Name)
		}

		tr.inputs[k] = v
	}

	// TODO: check if required fields are missing

	return nil
}

func (tr *TaskRun) Outputs() object.Object {
	var code strings.Builder

	code.WriteString("m := {}")

	for k, value := range tr.task.Outputs {
		if strings.HasPrefix(value, "${{") {
			expr := strings.TrimPrefix(value, "${{")
			expr = strings.TrimSuffix(expr, "}}")

			fmt.Fprintf(&code, `
m["%s"] = %s
`, k, expr)
		} else {
			fmt.Fprintf(&code, `
m["%s"] = "%s"
`, k, value)
		}
	}

	code.WriteString("\nm")

	out, err := tr.eval(code.String())
	if err != nil {
		fmt.Println(err)
	}

	return out
}

func (tr *TaskRun) setStepOutput(stepID string, obj object.Object) error {
	if stepID != "" {
		value, err := fromRisor(obj)
		if err != nil {
			return err
		}

		tr.stepOutputs[stepID] = value
	}

	return nil
}

func (tr *TaskRun) eval(expression string) (object.Object, error) {
	return risor.Eval(
		tr.ctx,
		expression,
		risor.WithGlobal("inputs", tr.inputs),
		risor.WithGlobal("steps", tr.stepOutputs),
		risor.WithGlobal("env", tr.Environ()),
	)
}

func (tr *TaskRun) exec(shell string, args []string) (object.Object, error) {
	result, err := risor.Eval(
		tr.ctx,
		"exec(shell, args, { env: env }).stdout",
		risor.WithGlobal("shell", shell),
		risor.WithGlobal("args", args),
		risor.WithGlobal("env", tr.Environ()),
	)

	fmt.Println(result)

	return result, err
}

func (tr *TaskRun) Environ() map[string]string {
	env := maps.Clone(tr.env)
	maps.Copy(env, helpers.TransformMapKeys(tr.inputs, func(key string) string {
		// TODO: very close to `utils.FormatEnvVar` in src/pkg/utils/utils.go#L73
		key = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(key, "_")
		key = strings.ToUpper(key)

		return strings.ToUpper("INPUT_" + key)
	}))
	return env
}

func (r *TaskRunner) getContext() (context.Context, error) {
	if _, ok := ros.GetOS(r.ctx); ok {
		// virtual OS already set by parent runner
		return r.ctx, nil
	}

	workdir, err := localfs.New(r.ctx, localfs.WithBase(r.workDir))

	if err != nil {
		return nil, err
	}

	return ros.WithOS(r.ctx, ros.NewVirtualOS(r.ctx,
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
	case *object.NilType:
	case *object.Bool:
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
	}

	return "", fmt.Errorf("unsupported output type: %T", value)
}

func getTaskName(includeKey, taskName string) string {
	if includeKey == "" {
		return taskName
	}

	return includeKey + ":" + taskName
}

func ToStep(a types.Action) *types.Step {
	return &types.Step{
		Env:     utils.EnvMap(a.Env),
		WorkDir: a.Dir,
		Cmd:     a.Cmd,
		Shell:   a.Shell,
		// Wait:    a.Wait,
		Uses:    a.TaskReference,
		With:    a.With,
		Timeout: a.MaxTotalSeconds,
		Retry:   a.MaxRetries,
	}
}
