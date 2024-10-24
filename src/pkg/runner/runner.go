// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package runner provides functions for running tasks in a tasks.yaml
package runner

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/config/lang"
	"github.com/defenseunicorns/maru-runner/src/message"
	"github.com/defenseunicorns/maru-runner/src/pkg/utils"
	"github.com/defenseunicorns/maru-runner/src/pkg/variables"
	"github.com/defenseunicorns/maru-runner/src/types"
	"github.com/defenseunicorns/pkg/helpers/v2"
)

// Runner holds the necessary data to run tasks from a tasks file
type Runner struct {
	TasksFile                       types.TasksFile
	ExistingTaskIncludeNameLocation map[string]string
	envFilePath                     string
	variableConfig                  *variables.VariableConfig[variables.ExtraVariableInfo]
	dryRun                          bool
	currStackSize                   int
}

// Run runs a task from tasks file
func Run(tasksFile types.TasksFile, taskName string, setVariables map[string]string, dryRun bool) error {
	if dryRun {
		message.SLog.Info("Dry-run has been set - only printing the commands that would run:")
	}

	// Populate the variables loaded in the root task file
	rootVariables := tasksFile.Variables
	rootVariableConfig := GetMaruVariableConfig()
	err := rootVariableConfig.PopulateVariables(rootVariables, setVariables)
	if err != nil {
		return err
	}

	// Check to see if running an included task directly
	tasksFile, taskName, err = loadIncludedTaskFile(tasksFile, taskName, rootVariableConfig.GetSetVariables())
	if err != nil {
		return err
	}

	// Populate the variables from the root and included file (if these are the same it will just use the same list)
	combinedVariables := helpers.MergeSlices(rootVariables, tasksFile.Variables, func(a, b variables.InteractiveVariable[variables.ExtraVariableInfo]) bool {
		return a.Name == b.Name
	})
	combinedVariableConfig := GetMaruVariableConfig()
	err = combinedVariableConfig.PopulateVariables(combinedVariables, setVariables)
	if err != nil {
		return err
	}

	// Create the runner client to execute the task file
	runner := Runner{
		TasksFile:                       tasksFile,
		ExistingTaskIncludeNameLocation: map[string]string{},
		variableConfig:                  combinedVariableConfig,
		dryRun:                          dryRun,
	}

	task, err := runner.getTask(taskName)
	if err != nil {
		return err
	}

	// Check that this task is a valid task we can call (i.e. has defaults for any inputs since those cannot be set on the CLI)
	if err := validateActionableTaskCall(task.Name, task.Inputs, nil); err != nil {
		return err
	}

	if err = runner.processTaskReferences(task, runner.TasksFile, setVariables); err != nil {
		return err
	}

	err = runner.executeTask(task, nil)
	return err
}

// GetMaruVariableConfig gets the variable configuration for Maru
func GetMaruVariableConfig() *variables.VariableConfig[variables.ExtraVariableInfo] {
	prompt := func(_ variables.InteractiveVariable[variables.ExtraVariableInfo]) (value string, err error) {
		return "", nil
	}
	return variables.New[variables.ExtraVariableInfo](prompt, message.SLog)
}

func (r *Runner) processIncludes(tasksFile types.TasksFile, setVariables map[string]string, action types.Action) error {
	if strings.Contains(action.TaskReference, ":") {
		taskReferenceName := strings.Split(action.TaskReference, ":")[0]
		for _, include := range tasksFile.Includes {
			if include[taskReferenceName] != "" {
				referencedIncludes := []map[string]string{include}
				err := r.importTasks(referencedIncludes, config.TaskFileLocation, setVariables)
				if err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}

func (r *Runner) importTasks(includes []map[string]string, currentFileLocation string, setVariables map[string]string) error {
	// iterate through includes, open the file, and unmarshal it into a Task
	var includeKey string
	var includeLocation string
	for _, include := range includes {
		if len(include) > 1 {
			return fmt.Errorf("included item %s must have only one key", include)
		}
		// grab first and only value from include map
		for k, v := range include {
			includeKey = k
			includeLocation = v
			break
		}

		includeLocation = utils.TemplateString(r.variableConfig.GetSetVariables(), includeLocation)

		absIncludeFileLocation, tasksFile, err := loadIncludeTask(currentFileLocation, includeLocation)
		if err != nil {
			return fmt.Errorf("unable to read included file: %w", err)
		}
		// If we arrive here we assume this was a new include due to the later check
		r.ExistingTaskIncludeNameLocation[includeKey] = absIncludeFileLocation

		// prefix task names and actions with the includes key
		for i, t := range tasksFile.Tasks {
			tasksFile.Tasks[i].Name = includeKey + ":" + t.Name
			if len(tasksFile.Tasks[i].Actions) > 0 {
				for j, a := range tasksFile.Tasks[i].Actions {
					if a.TaskReference != "" && !strings.Contains(a.TaskReference, ":") {
						tasksFile.Tasks[i].Actions[j].TaskReference = includeKey + ":" + a.TaskReference
					}
				}
			}
		}

		r.TasksFile.Tasks = append(r.TasksFile.Tasks, tasksFile.Tasks...)

		r.mergeVariablesFromIncludedTask(tasksFile)

		// recursively import tasks from included files
		if tasksFile.Includes != nil {
			newIncludes := []map[string]string{}
			var newIncludeKey string
			var newIncludeLocation string
			for _, newInclude := range tasksFile.Includes {
				for k, v := range newInclude {
					newIncludeKey = k
					newIncludeLocation = v
					break
				}
				if existingLocation, exists := r.ExistingTaskIncludeNameLocation[newIncludeKey]; !exists {
					newIncludes = append(newIncludes, map[string]string{newIncludeKey: newIncludeLocation})
				} else {
					newAbsIncludeFileLocation, err := includeTaskAbsLocation(absIncludeFileLocation, newIncludeLocation)
					if err != nil {
						return err
					}
					if existingLocation != newAbsIncludeFileLocation {
						return fmt.Errorf("task include %q attempted to be redefined from %q to %q", includeKey, existingLocation, newAbsIncludeFileLocation)
					}
				}
			}
			if err := r.importTasks(newIncludes, absIncludeFileLocation, setVariables); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) mergeVariablesFromIncludedTask(tasksFile types.TasksFile) {
	// grab variables from included file
	for _, v := range tasksFile.Variables {
		if _, ok := r.variableConfig.GetSetVariable(v.Name); !ok {
			r.variableConfig.SetVariable(v.Name, v.Default, v.Pattern, v.Extra)
		}
	}
}

func loadIncludedTaskFile(taskFile types.TasksFile, taskName string, setVariables variables.SetVariableMap[variables.ExtraVariableInfo]) (types.TasksFile, string, error) {
	// Check if running task directly from included task file
	includedTask := strings.Split(taskName, ":")
	if len(includedTask) == 2 {
		includeName := includedTask[0]
		includeTaskName := includedTask[1]
		// Get referenced include file
		for _, includes := range taskFile.Includes {
			if includeFileLocation, ok := includes[includeName]; ok {
				includeFileLocation = utils.TemplateString(setVariables, includeFileLocation)

				absIncludeFileLocation, includedTasksFile, err := loadIncludeTask(config.TaskFileLocation, includeFileLocation)
				config.TaskFileLocation = absIncludeFileLocation
				return includedTasksFile, includeTaskName, err
			}
		}
	} else if len(includedTask) > 2 {
		return taskFile, taskName, fmt.Errorf("invalid task name: %s", taskName)
	}
	return taskFile, taskName, nil
}

func includeTaskAbsLocation(currentFileLocation, includeFileLocation string) (string, error) {
	var absIncludeFileLocation string

	if !helpers.IsURL(includeFileLocation) {
		if helpers.IsURL(currentFileLocation) {
			currentURL, err := url.Parse(currentFileLocation)
			if err != nil {
				return absIncludeFileLocation, err
			}
			currentURL.Path = path.Join(path.Dir(currentURL.Path), includeFileLocation)
			absIncludeFileLocation = currentURL.String()
		} else {
			// Calculate the full path for local (and most remote) references
			absIncludeFileLocation = filepath.Join(filepath.Dir(currentFileLocation), includeFileLocation)
		}
	} else {
		absIncludeFileLocation = includeFileLocation
	}
	return absIncludeFileLocation, nil
}

func loadIncludeTask(currentFileLocation, includeFileLocation string) (string, types.TasksFile, error) {
	var localPath string
	var includedTasksFile types.TasksFile

	absIncludeFileLocation, err := includeTaskAbsLocation(currentFileLocation, includeFileLocation)
	if err != nil {
		return absIncludeFileLocation, includedTasksFile, err
	}

	// If the file is in fact a URL we need to download and load the YAML
	if helpers.IsURL(absIncludeFileLocation) {
		// If file is a url download it to a tmp directory
		tmpDir, err := utils.MakeTempDir(config.TempDirectory)
		if err != nil {
			return absIncludeFileLocation, includedTasksFile, fmt.Errorf("error creating %s: %w", tmpDir, err)
		}

		// Remove tmpDir, but not until tasks have been loaded
		defer os.RemoveAll(tmpDir)
		localPath = filepath.Join(tmpDir, filepath.Base(absIncludeFileLocation))
		if err := utils.DownloadToFile(absIncludeFileLocation, localPath); err != nil {
			return absIncludeFileLocation, includedTasksFile, fmt.Errorf(lang.ErrDownloading, absIncludeFileLocation, err)
		}
	} else {
		localPath = absIncludeFileLocation
	}

	// Set TasksFile to include task file
	err = utils.ReadYaml(localPath, &includedTasksFile)
	return absIncludeFileLocation, includedTasksFile, err
}

func (r *Runner) getTask(taskName string) (types.Task, error) {
	for _, task := range r.TasksFile.Tasks {
		if task.Name == taskName {
			return task, nil
		}
	}
	return types.Task{}, fmt.Errorf("task name %s not found", taskName)
}

func (r *Runner) executeTask(task types.Task, withs map[string]string) error {
	if r.currStackSize > config.MaxStack {
		return fmt.Errorf("task looping exceeded max configured task stack of %d", config.MaxStack)
	}

	r.currStackSize++
	defer func() {
		r.currStackSize--
	}()

	defaultEnv := []string{}
	for name, inputParam := range task.Inputs {
		d := inputParam.Default
		if d == "" {
			continue
		}
		defaultEnv = append(defaultEnv, utils.FormatEnvVar(name, d))
	}

	// load the tasks env file into the runner, can override previous task's env files
	if task.EnvPath != "" {
		r.envFilePath = task.EnvPath
	}

	for _, action := range task.Actions {
		action.Env = utils.MergeEnv(action.Env, defaultEnv)
		if err := r.performAction(action, withs, task.Inputs); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) processTaskReferences(task types.Task, tasksFile types.TasksFile, setVariables map[string]string) error {
	if r.currStackSize > config.MaxStack {
		return fmt.Errorf("task looping exceeded max configured task stack of %d", config.MaxStack)
	}

	r.currStackSize++
	defer func() {
		r.currStackSize--
	}()

	// Filtering unique task actions allows for rerunning tasks in the same execution
	uniqueTaskActions := getUniqueTaskActions(task.Actions)
	for _, action := range uniqueTaskActions {
		if r.processAction(task, action) {
			// process includes for action, which will import all tasks for include file
			if err := r.processIncludes(tasksFile, setVariables, action); err != nil {
				return err
			}

			newTask, err := r.getTask(action.TaskReference)
			if err != nil {
				return err
			}
			if err = r.processTaskReferences(newTask, tasksFile, setVariables); err != nil {
				return err
			}
		}
	}
	return nil
}
