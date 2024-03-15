// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

// Package runner provides functions for running tasks in a tasks.yaml
package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/pkg/utils"
	"github.com/defenseunicorns/maru-runner/src/types"
	zarfConfig "github.com/defenseunicorns/zarf/src/config"
	"github.com/defenseunicorns/zarf/src/config/lang"
	"github.com/defenseunicorns/zarf/src/pkg/message"
	zarfUtils "github.com/defenseunicorns/zarf/src/pkg/utils"
	"github.com/defenseunicorns/zarf/src/pkg/utils/helpers"
	zarfTypes "github.com/defenseunicorns/zarf/src/types"
	"github.com/mholt/archiver/v3"
)

// Runner holds the necessary data to run tasks from a tasks file
type Runner struct {
	TemplateMap map[string]*zarfUtils.TextTemplate
	TasksFile   types.TasksFile
	TaskNameMap map[string]bool
	envFilePath string
	runner      ActionRunner
}

// NewRunner returns a new Runner.
func NewRunner(runner ActionRunner, tasksFile types.TasksFile) *Runner {
	return &Runner{
		runner:      runner,
		TasksFile:   tasksFile,
		TaskNameMap: map[string]bool{},
		TemplateMap: map[string]*zarfUtils.TextTemplate{},
	}
}

// Run runs a task from tasks file
func (r *Runner) Run(taskName string, setVariables map[string]string) error {

	// Check to see if running an included task directly
	includeTaskName, err := r.loadIncludedTaskFile(taskName)
	if err != nil {
		return err
	}
	// if running an included task directly, update the task name
	if len(includeTaskName) > 0 {
		taskName = includeTaskName
	}

	task, err := r.getTask(taskName)
	if err != nil {
		return err
	}

	// populate after getting task in case of calling included task directly
	templateMap := utils.PopulateTemplateMap(r.TasksFile.Variables, setVariables)
	r.TemplateMap = templateMap

	// can't call a task directly from the CLI if it has inputs
	if task.Inputs != nil {
		return fmt.Errorf("task '%s' contains 'inputs' and cannot be called directly by the CLI", taskName)
	}

	if err = r.checkForTaskLoops(task, r.TasksFile, setVariables); err != nil {
		return err
	}

	err = r.executeTask(task)
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

		includeFilename = utils.TemplateString(r.TemplateMap, includeFilename)

		var tasksFile types.TasksFile
		var includePath string
		// check if included file is a url
		if helpers.IsURL(includeFilename) {
			// If file is a url download it to a tmp directory
			tmpDir, err := zarfUtils.MakeTempDir(config.TempDirectory)
			defer os.RemoveAll(tmpDir)
			if err != nil {
				return err
			}
			includePath = filepath.Join(tmpDir, filepath.Base(includeFilename))
			if err := zarfUtils.DownloadToFile(includeFilename, includePath, ""); err != nil {
				return fmt.Errorf(lang.ErrDownloading, includeFilename, err.Error())
			}
		} else {
			includePath = filepath.Join(dir, includeFilename)
		}

		if err := zarfUtils.ReadYaml(includePath, &tasksFile); err != nil {
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

		r.processTemplateMapVariables(setVariables, tasksFile)

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

func (r *Runner) processTemplateMapVariables(setVariables map[string]string, tasksFile types.TasksFile) {
	// grab variables from included file
	for _, v := range tasksFile.Variables {
		r.TemplateMap["${"+v.Name+"}"] = &zarfUtils.TextTemplate{
			Sensitive:  v.Sensitive,
			AutoIndent: v.AutoIndent,
			Type:       v.Type,
			Value:      v.Default,
		}
	}

	// merge variables with setVariables
	setVariablesTemplateMap := make(map[string]*zarfUtils.TextTemplate)
	for name, value := range setVariables {
		setVariablesTemplateMap[fmt.Sprintf("${%s}", name)] = &zarfUtils.TextTemplate{
			Value: value,
		}
	}

	r.TemplateMap = helpers.MergeMap[*zarfUtils.TextTemplate](r.TemplateMap, setVariablesTemplateMap)
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
		templateMap := utils.PopulateTemplateMap(r.TasksFile.Variables, config.SetRunnerVariables)
		includeFileLocation = utils.TemplateString(templateMap, includeFileLocation)
	}
	// check if included file is a url
	if helpers.IsURL(includeFileLocation) {
		// If file is a url download it to a tmp directory
		tmpDir, err := zarfUtils.MakeTempDir(zarfConfig.CommonOptions.TempDirectory)
		if err != nil {
			message.Fatalf(err, "error creating %s", tmpDir)
		}
		// Remove tmpDir, but not until tasks have been loaded
		defer os.RemoveAll(tmpDir)
		fullPath = filepath.Join(tmpDir, filepath.Base(includeFileLocation))
		if err := zarfUtils.DownloadToFile(includeFileLocation, fullPath, ""); err != nil {
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
	err := zarfUtils.ReadYaml(fullPath, &tasksFile)
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

func (r *Runner) placeFiles(files []zarfTypes.ZarfFile) error {
	for _, file := range files {
		// template file.Source and file.Target
		srcFile := utils.TemplateString(r.TemplateMap, file.Source)
		targetFile := utils.TemplateString(r.TemplateMap, file.Target)

		// get current directory
		workingDir, err := os.Getwd()
		if err != nil {
			return err
		}
		dest := filepath.Join(workingDir, targetFile)
		destDir := filepath.Dir(dest)

		if helpers.IsURL(srcFile) {
			// If file is a url download it
			if err := zarfUtils.DownloadToFile(srcFile, dest, ""); err != nil {
				return fmt.Errorf(lang.ErrDownloading, srcFile, err.Error())
			}
		} else {
			// If file is not a url copy it
			if err := zarfUtils.CreatePathAndCopy(srcFile, dest); err != nil {
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
				if err := zarfUtils.SHAsMatch(file.ExtractPath, file.Shasum); err != nil {
					return err
				}
			} else {
				if err := zarfUtils.SHAsMatch(dest, file.Shasum); err != nil {
					return err
				}
			}
		}

		r.templateTextFilesWithVars(dest)

		// if executable make file executable
		if file.Executable || zarfUtils.IsDir(dest) {
			_ = os.Chmod(dest, 0700)
		} else {
			_ = os.Chmod(dest, 0600)
		}

		// if symlinks create them
		for _, link := range file.Symlinks {
			// Try to remove the filepath if it exists
			_ = os.RemoveAll(link)
			// Make sure the parent directory exists
			_ = zarfUtils.CreateParentDirectory(link)
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
	if zarfUtils.IsDir(dest) {
		files, _ := zarfUtils.RecursiveFileList(dest, nil, false)
		fileList = append(fileList, files...)
	} else {
		fileList = append(fileList, dest)
	}
	for _, subFile := range fileList {
		// Check if the file looks like a text file
		isText, err := zarfUtils.IsTextFile(subFile)
		if err != nil {
			fmt.Printf("unable to determine if file %s is a text file: %s", subFile, err)
		}

		// If the file is a text file, template it
		if isText {
			if err := zarfUtils.ReplaceTextTemplate(subFile, r.TemplateMap, nil, `\$\{[A-Z0-9_]+\}`); err != nil {
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
