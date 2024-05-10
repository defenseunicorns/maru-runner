// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package runner provides functions for running tasks in a tasks.yaml
package runner

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/config/lang"
	"github.com/defenseunicorns/maru-runner/src/pkg/utils"
	"github.com/defenseunicorns/maru-runner/src/types"
	"github.com/defenseunicorns/pkg/helpers"
	"github.com/defenseunicorns/pkg/variables"
	"github.com/defenseunicorns/zarf/src/pkg/message"
	"github.com/mholt/archiver/v3"
)

func maruVariableConfig() *variables.VariableConfig {
	prompt := func(_ variables.InteractiveVariable) (value string, err error) { return "", nil }
	return variables.New("", map[string]string{}, prompt, slog.New(message.ZarfHandler{}))
}

// Runner holds the necessary data to run tasks from a tasks file
type Runner struct {
	TasksFile      types.TasksFile
	TaskNameMap    map[string]bool
	envFilePath    string
	variableConfig *variables.VariableConfig
}

// Run runs a task from tasks file
func Run(tasksFile types.TasksFile, taskName string, setVariables map[string]string) error {
	runner := Runner{
		TasksFile:      tasksFile,
		TaskNameMap:    map[string]bool{},
		variableConfig: maruVariableConfig(),
	}

	// Populate the variables loaded in the task file
	runner.variableConfig.PopulateVariables(runner.TasksFile.Variables, setVariables)

	// Check to see if running an included task directly
	includeTaskName, err := runner.loadIncludedTaskFile(taskName)
	if err != nil {
		return err
	}
	// if running an included task directly, update the task name
	if len(includeTaskName) > 0 {
		taskName = includeTaskName
	}

	task, err := runner.getTask(taskName)
	if err != nil {
		return err
	}

	// can't call a task directly from the CLI if it has inputs
	if task.Inputs != nil {
		return fmt.Errorf("task '%s' contains 'inputs' and cannot be called directly by the CLI", taskName)
	}

	if err = runner.checkForTaskLoops(task, runner.TasksFile, setVariables); err != nil {
		return err
	}

	err = runner.executeTask(task)
	return err
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

func (r *Runner) importTasks(includes []map[string]string, dir string, setVariables map[string]string) error {
	// iterate through includes, open the file, and unmarshal it into a Task
	var includeFilenameKey string
	var includeFilename string
	dir = filepath.Dir(dir)
	for _, include := range includes {
		if len(include) > 1 {
			return fmt.Errorf("included item %s must have only one key", include)
		}
		// grab first and only value from include map
		for k, v := range include {
			includeFilenameKey = k
			includeFilename = v
			break
		}

		includeFilename = utils.TemplateString(r.variableConfig.GetSetVariables(), includeFilename)

		var tasksFile types.TasksFile
		var includePath string
		// check if included file is a url
		if helpers.IsURL(includeFilename) {
			// If file is a url download it to a tmp directory
			tmpDir, err := utils.MakeTempDir(config.TempDirectory)
			defer os.RemoveAll(tmpDir)
			if err != nil {
				return err
			}
			includePath = filepath.Join(tmpDir, filepath.Base(includeFilename))
			if err := utils.DownloadToFile(includeFilename, includePath); err != nil {
				return fmt.Errorf(lang.ErrDownloading, includeFilename, err.Error())
			}
		} else {
			includePath = filepath.Join(dir, includeFilename)
		}

		if err := utils.ReadYaml(includePath, &tasksFile); err != nil {
			return fmt.Errorf("unable to read included file %s: %w", includePath, err)
		}

		// prefix task names and actions with the includes key
		for i, t := range tasksFile.Tasks {
			tasksFile.Tasks[i].Name = includeFilenameKey + ":" + t.Name
			if len(tasksFile.Tasks[i].Actions) > 0 {
				for j, a := range tasksFile.Tasks[i].Actions {
					if a.TaskReference != "" && !strings.Contains(a.TaskReference, ":") {
						tasksFile.Tasks[i].Actions[j].TaskReference = includeFilenameKey + ":" + a.TaskReference
					}
				}
			}
		}
		err := r.checkProcessedTasksForLoops(tasksFile)
		if err != nil {
			return err
		}

		r.TasksFile.Tasks = append(r.TasksFile.Tasks, tasksFile.Tasks...)

		r.processTemplateMapVariables(tasksFile)

		// recursively import tasks from included files
		if tasksFile.Includes != nil {
			if err := r.importTasks(tasksFile.Includes, includePath, setVariables); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) checkProcessedTasksForLoops(tasksFile types.TasksFile) error {
	// The following for loop protects against task loops. Makes sure the task being added hasn't already been processed
	for _, taskToAdd := range tasksFile.Tasks {
		for _, currentTasks := range r.TasksFile.Tasks {
			if taskToAdd.Name == currentTasks.Name {
				return fmt.Errorf("task loop detected, ensure no cyclic loops in tasks or includes files")
			}
		}
	}
	return nil
}

func (r *Runner) processTemplateMapVariables(tasksFile types.TasksFile) {
	// grab variables from included file
	for _, v := range tasksFile.Variables {
		if _, ok := r.variableConfig.GetSetVariable(v.Name); !ok {
			r.variableConfig.SetVariable(v.Name, v.Default, v.Sensitive, v.AutoIndent, v.Type)
		}
	}
}

func (r *Runner) loadIncludedTaskFile(taskName string) (string, error) {
	// Check if running task directly from included task file
	includedTask := strings.Split(taskName, ":")
	if len(includedTask) == 2 {
		includeName := includedTask[0]
		includeTaskName := includedTask[1]
		// Get referenced include file
		for _, includes := range r.TasksFile.Includes {
			if includeFileLocation, ok := includes[includeName]; ok {
				return r.loadIncludeTask(includeFileLocation, includeTaskName)
			}
		}
	} else if len(includedTask) > 2 {
		return "", fmt.Errorf("invalid task name: %s", taskName)
	}
	return "", nil
}

func (r *Runner) loadIncludeTask(includeFileLocation string, includeTaskName string) (string, error) {
	var fullPath string
	templatePattern := `\${[^}]+}`
	re := regexp.MustCompile(templatePattern)

	// check for templated variables in includeFileLocation value
	if re.MatchString(includeFileLocation) {
		includeFileLocation = utils.TemplateString(r.variableConfig.GetSetVariables(), includeFileLocation)
	}
	// check if included file is a url
	if helpers.IsURL(includeFileLocation) {
		// If file is a url download it to a tmp directory
		tmpDir, err := utils.MakeTempDir(config.TempDirectory)
		if err != nil {
			message.Fatalf(err, "error creating %s", tmpDir)
		}
		// Remove tmpDir, but not until tasks have been loaded
		defer os.RemoveAll(tmpDir)
		fullPath = filepath.Join(tmpDir, filepath.Base(includeFileLocation))
		if err := utils.DownloadToFile(includeFileLocation, fullPath); err != nil {
			message.Fatalf(lang.ErrDownloading, includeFileLocation, err.Error())
		}
	} else {
		// set include path based on task file location
		fullPath = filepath.Join(filepath.Dir(config.TaskFileLocation), includeFileLocation)
	}
	// update config.TaskFileLocation which gets used globally
	config.TaskFileLocation = fullPath

	// Set TasksFile to include task file
	r.TasksFile = loadTasksFileFromPath(fullPath)
	taskName := includeTaskName
	return taskName, nil
}

func loadTasksFileFromPath(fullPath string) types.TasksFile {
	var tasksFile types.TasksFile
	// get included TasksFile
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		message.Fatalf(err, "%s not found", config.TaskFileLocation)
	}
	err := utils.ReadYaml(fullPath, &tasksFile)
	if err != nil {
		message.Fatalf(err, "Cannot unmarshal %s", config.TaskFileLocation)
	}
	return tasksFile
}

func (r *Runner) getTask(taskName string) (types.Task, error) {
	for _, task := range r.TasksFile.Tasks {
		if task.Name == taskName {
			return task, nil
		}
	}
	return types.Task{}, fmt.Errorf("task name %s not found", taskName)
}

func (r *Runner) executeTask(task types.Task) error {
	if len(task.Files) > 0 {
		if err := r.placeFiles(task.Files); err != nil {
			return err
		}
	}

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
		if err := r.performAction(action); err != nil {
			return err
		}
	}
	return nil
}

// TODO (@WSTARR) Evaluate if this is used / useful
func (r *Runner) placeFiles(files []types.File) error {
	for _, file := range files {
		// template file.Source and file.Target
		srcFile := utils.TemplateString(r.variableConfig.GetSetVariables(), file.Source)
		targetFile := utils.TemplateString(r.variableConfig.GetSetVariables(), file.Target)

		// get current directory
		workingDir, err := os.Getwd()
		if err != nil {
			return err
		}
		dest := filepath.Join(workingDir, targetFile)
		destDir := filepath.Dir(dest)

		if helpers.IsURL(srcFile) {
			// If file is a url download it
			if err := utils.DownloadToFile(srcFile, dest); err != nil {
				return fmt.Errorf(lang.ErrDownloading, srcFile, err.Error())
			}
		} else {
			// If file is not a url copy it
			if err := helpers.CreatePathAndCopy(srcFile, dest); err != nil {
				return fmt.Errorf("unable to copy file %s: %w", srcFile, err)
			}

		}
		// If file has extract path extract it
		if file.ExtractPath != "" {
			_ = os.RemoveAll(file.ExtractPath)
			err = archiver.Extract(dest, file.ExtractPath, destDir)
			if err != nil {
				return fmt.Errorf(lang.ErrFileExtract, file.ExtractPath, srcFile, err.Error())
			}
		}

		// if shasum is specified check it
		if file.Shasum != "" {
			if file.ExtractPath != "" {
				if err := helpers.SHAsMatch(file.ExtractPath, file.Shasum); err != nil {
					return err
				}
			} else {
				if err := helpers.SHAsMatch(dest, file.Shasum); err != nil {
					return err
				}
			}
		}

		r.templateTextFilesWithVars(dest)

		// if executable make file executable
		if file.Executable || helpers.IsDir(dest) {
			_ = os.Chmod(dest, 0700)
		} else {
			_ = os.Chmod(dest, 0600)
		}

		// if symlinks create them
		for _, link := range file.Symlinks {
			// Try to remove the filepath if it exists
			_ = os.RemoveAll(link)
			// Make sure the parent directory exists
			_ = helpers.CreateParentDirectory(link)
			// Create the symlink
			err := os.Symlink(targetFile, link)
			if err != nil {
				return fmt.Errorf("unable to create symlink %s->%s: %w", link, targetFile, err)
			}
		}
	}
	return nil
}

func (r *Runner) templateTextFilesWithVars(dest string) {
	fileList := []string{}
	if helpers.IsDir(dest) {
		files, _ := helpers.RecursiveFileList(dest, nil, false)
		fileList = append(fileList, files...)
	} else {
		fileList = append(fileList, dest)
	}
	for _, subFile := range fileList {
		// Check if the file looks like a text file
		isText, err := helpers.IsTextFile(subFile)
		if err != nil {
			fmt.Printf("unable to determine if file %s is a text file: %s", subFile, err)
		}

		// If the file is a text file, template it
		if isText {
			// TODO (@WSTARR) - I broke this because the delims are not configurable anymore
			if err := r.variableConfig.ReplaceTextTemplate(subFile); err != nil {
				message.Fatalf(err, "unable to template file %s", subFile)
			}
		}
	}
}

func (r *Runner) checkForTaskLoops(task types.Task, tasksFile types.TasksFile, setVariables map[string]string) error {
	// Filtering unique task actions allows for rerunning tasks in the same execution
	uniqueTaskActions := getUniqueTaskActions(task.Actions)
	for _, action := range uniqueTaskActions {
		if r.processAction(task, action) {
			// process includes for action, which will import all tasks for include file
			if err := r.processIncludes(tasksFile, setVariables, action); err != nil {
				return err
			}

			exists := r.TaskNameMap[action.TaskReference]
			if exists {
				return fmt.Errorf("task loop detected, ensure no cyclic loops in tasks or includes files")
			}
			r.TaskNameMap[action.TaskReference] = true
			newTask, err := r.getTask(action.TaskReference)
			if err != nil {
				return err
			}
			if err = r.checkForTaskLoops(newTask, tasksFile, setVariables); err != nil {
				return err
			}
		}
		// Clear map once we get to a task that doesn't call another task
		clear(r.TaskNameMap)
	}
	return nil
}
