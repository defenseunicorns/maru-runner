// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package runner provides functions for running tasks in a tasks.yaml
package runner

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/message"
	"github.com/defenseunicorns/maru-runner/src/pkg/utils"
	"github.com/defenseunicorns/maru-runner/src/pkg/variables"
	"github.com/defenseunicorns/maru-runner/src/types"
	"github.com/defenseunicorns/pkg/helpers/v2"
	"github.com/hashicorp/go-getter"
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
	for _, include := range includes {
		if len(include) > 1 {
			return fmt.Errorf("included item %s must have only one key", include)
		}
		// grab first and only value from include map
		var includeKey, includeLocation string
		for k, v := range include {
			includeKey = k
			includeLocation = v
			break
		}

		// Apply variable substitution to includeLocation
		includeLocation = utils.TemplateString(r.variableConfig.GetSetVariables(), includeLocation)

		absIncludeFileLocation, tasksFile, err := loadIncludeTask(currentFileLocation, includeLocation, includeKey)
		if err != nil {
			return fmt.Errorf("unable to read included file: %w", err)
		}
		// If we arrive here we assume this was a new include due to the later check
		r.ExistingTaskIncludeNameLocation[includeKey] = absIncludeFileLocation

		// Prefix task names and actions with the includes key
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

		// Recursively import tasks from included files
		if tasksFile.Includes != nil {
			var newIncludes []map[string]string
			for _, newInclude := range tasksFile.Includes {
				var newIncludeKey, newIncludeLocation string
				for k, v := range newInclude {
					newIncludeKey = k
					newIncludeLocation = v
					break
				}

				// Apply variable substitution to newIncludeLocation
				newIncludeLocation = utils.TemplateString(r.variableConfig.GetSetVariables(), newIncludeLocation)

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

				absIncludeFileLocation, includedTasksFile, err := loadIncludeTask(config.TaskFileLocation, includeFileLocation, includeName)
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

	if strings.HasPrefix(includeFileLocation, "git::") {
		// Treat it as a URL
		absIncludeFileLocation = includeFileLocation
	} else if !helpers.IsURL(includeFileLocation) {
		if helpers.IsURL(currentFileLocation) {
			// currentFileLocation is a URL
			currentURL, err := url.Parse(currentFileLocation)
			if err != nil {
				return "", err
			}
			currentURL.Path = path.Join(path.Dir(currentURL.Path), includeFileLocation)
			absIncludeFileLocation = currentURL.String()
		} else {
			// Both currentFileLocation and includeFileLocation are local paths
			absPath, err := filepath.Abs(filepath.Join(filepath.Dir(currentFileLocation), includeFileLocation))
			if err != nil {
				return "", err
			}
			absIncludeFileLocation = absPath
		}
	} else {
		// includeFileLocation is a URL
		absIncludeFileLocation = includeFileLocation
	}
	return absIncludeFileLocation, nil
}

func fetchIncludedTasks(ctx context.Context, source string, destination string) error {
	var client *getter.Client

	if strings.HasPrefix(source, "git::") {
		// Handle Git includes
		modifiedSource := source

		// Parse the source URL
		u, err := url.Parse(strings.TrimPrefix(source, "git::"))
		if err != nil {
			return fmt.Errorf("invalid source URL: %w", err)
		}

		// Extract repository URL and subdirectory
		repoURL := u.Scheme + "://" + u.Host + strings.SplitN(u.Path, "//", 2)[0]

		// Extract the subdirectory path (after the double slash)
		subPath := ""
		if strings.Contains(u.Path, "//") {
			subPath = strings.SplitN(u.Path, "//", 2)[1]
		}

		// Remove the file name from subPath to get the directory
		if subPath != "" {
			subPathDir := filepath.Dir(subPath)
			repoURL += "//" + subPathDir
		}

		// Reconstruct the modified source URL
		modifiedSource = "git::" + repoURL
		if u.RawQuery != "" {
			modifiedSource += "?" + u.RawQuery
		}

		client = &getter.Client{
			Ctx:  ctx,
			Src:  modifiedSource,
			Dst:  destination,
			Pwd:  ".", // Current working directory
			Mode: getter.ClientModeDir,
		}
	} else if helpers.IsURL(source) {
		// Handle HTTP/HTTPS URLs
		client = &getter.Client{
			Ctx:  ctx,
			Src:  source,
			Dst:  destination,
			Pwd:  ".",
			Mode: getter.ClientModeFile,
		}
	} else {
		// For local files, we should not call fetchIncludedTasks
		return fmt.Errorf("source %s is a local file, fetchIncludedTasks should not be called", source)
	}

	if err := client.Get(); err != nil {
		return fmt.Errorf("error fetching source %s: %w", source, err)
	}

	return nil
}

func loadIncludeTask(currentFileLocation, includeFileLocation, includeKey string) (string, types.TasksFile, error) {
	var includedTasksFile types.TasksFile

	absIncludeFileLocation, err := includeTaskAbsLocation(currentFileLocation, includeFileLocation)
	if err != nil {
		return absIncludeFileLocation, includedTasksFile, err
	}

	if strings.HasPrefix(includeFileLocation, "git::") {
		// Handle Git includes
		tempDir, err := utils.MakeTempDir(config.TempDirectory)
		if err != nil {
			return absIncludeFileLocation, includedTasksFile, fmt.Errorf("error creating temp directory: %w", err)
		}
		defer os.RemoveAll(tempDir)

		// Generate a unique destination path using the include key
		destination := filepath.Join(tempDir, "included_task_"+includeKey)

		if err := fetchIncludedTasks(context.Background(), absIncludeFileLocation, destination); err != nil {
			return absIncludeFileLocation, includedTasksFile, fmt.Errorf("failed to fetch included tasks: %w", err)
		}

		// Extract the file path from the original includeFileLocation
		u, err := url.Parse(strings.TrimPrefix(includeFileLocation, "git::"))
		if err != nil {
			return absIncludeFileLocation, includedTasksFile, fmt.Errorf("invalid include file URL: %w", err)
		}

		// Get the subdirectory and file path
		subPath := ""
		if strings.Contains(u.Path, "//") {
			subPath = strings.SplitN(u.Path, "//", 2)[1]
		}

		// Since go-getter flattens the directory structure, adjust the localPath
		fileName := filepath.Base(subPath)
		localPath := filepath.Join(destination, fileName)

		// Check if the file exists
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			return absIncludeFileLocation, includedTasksFile, fmt.Errorf("task file %s not found in the included directory", localPath)
		}

		// Read the tasks file from localPath
		err = utils.ReadYaml(localPath, &includedTasksFile)
		return absIncludeFileLocation, includedTasksFile, err

	} else if !helpers.IsURL(absIncludeFileLocation) {
		// Handle local includes
		localPath := absIncludeFileLocation

		// Check if the file exists
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			return absIncludeFileLocation, includedTasksFile, fmt.Errorf("task file %s not found", localPath)
		}

		// Read the tasks file directly from the local path
		err = utils.ReadYaml(localPath, &includedTasksFile)
		return absIncludeFileLocation, includedTasksFile, err

	} else {
		// Handle remote URL includes (non-Git URLs)
		tempDir, err := utils.MakeTempDir(config.TempDirectory)
		if err != nil {
			return absIncludeFileLocation, includedTasksFile, fmt.Errorf("error creating temp directory: %w", err)
		}
		defer os.RemoveAll(tempDir)

		// Destination is the file path where the file will be saved
		destination := filepath.Join(tempDir, "included_task_"+includeKey+".yaml")

		if err := fetchIncludedTasks(context.Background(), absIncludeFileLocation, destination); err != nil {
			return absIncludeFileLocation, includedTasksFile, fmt.Errorf("failed to fetch included tasks: %w", err)
		}

		localPath := destination

		// Check if the file exists
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			return absIncludeFileLocation, includedTasksFile, fmt.Errorf("task file %s not found in the included directory", localPath)
		}

		// Read the tasks file from localPath
		err = utils.ReadYaml(localPath, &includedTasksFile)
		return absIncludeFileLocation, includedTasksFile, err
	}
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
