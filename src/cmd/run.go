// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package cmd contains the CLI commands for maru.
package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/config/lang"
	"github.com/defenseunicorns/maru-runner/src/message"
	"github.com/defenseunicorns/maru-runner/src/pkg/runner"
	"github.com/defenseunicorns/maru-runner/src/pkg/utils"
	"github.com/defenseunicorns/maru-runner/src/types"
	"github.com/defenseunicorns/pkg/helpers/v2"
	goyaml "github.com/goccy/go-yaml"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// listTasks is a flag to print available tasks in a TaskFileLocation (no includes)
var listTasks bool

// listAllTasks is a flag to print available tasks in a TaskFileLocation
var listAllTasks bool

// dryRun is a flag to only load / validate tasks without running commands
var dryRun bool

// setRunnerVariables provides a map of set variables from the command line
var setRunnerVariables map[string]string

var runCmd = &cobra.Command{
	Use: "run",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		exitOnInterrupt()
		cliSetup()
	},
	Short:             lang.RootCmdShort,
	ValidArgsFunction: ListAutoCompleteTasks,
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("accepts 0 or 1 arg(s), received %d", len(args))
		}
		return nil
	},
	Run: func(_ *cobra.Command, args []string) {
		var tasksFile types.TasksFile

		err := utils.ReadYaml(config.TaskFileLocation, &tasksFile)
		if err != nil {
			message.Fatalf(err, "Failed to open file: %s", err.Error())
		}

		// ensure vars are uppercase
		setRunnerVariables = helpers.TransformMapKeys(setRunnerVariables, strings.ToUpper)

		// set any env vars that come from the environment (taking MARU_ over VENDOR_)
		for _, variable := range tasksFile.Variables {
			if _, ok := setRunnerVariables[variable.Name]; !ok {
				if value := os.Getenv(fmt.Sprintf("%s_%s", strings.ToUpper(config.EnvPrefix), variable.Name)); value != "" {
					setRunnerVariables[variable.Name] = value
				} else if config.VendorPrefix != "" {
					if value := os.Getenv(fmt.Sprintf("%s_%s", strings.ToUpper(config.VendorPrefix), variable.Name)); value != "" {
						setRunnerVariables[variable.Name] = value
					}
				}
			}
		}

		if listTasks || listAllTasks {
			rows := [][]string{
				{"Name", "Description"},
			}
			for _, task := range tasksFile.Tasks {
				rows = append(rows, []string{task.Name, task.Description})
			}

			// If ListAllTasks, add tasks from included files
			if listAllTasks {
				err = listTasksFromIncludes(&rows, tasksFile)
				if err != nil {
					message.Fatalf(err, "Cannot list tasks: %s", err.Error())
				}
			}

			err := pterm.DefaultTable.WithHasHeader().WithData(rows).Render()
			if err != nil {
				message.Fatalf(err, "Error listing tasks: %s", err.Error())
			}

			return
		}

		taskName := "default"
		if len(args) > 0 {
			taskName = args[0]
		}
		if err := runner.Run(tasksFile, taskName, setRunnerVariables, dryRun); err != nil {
			message.Fatalf(err, "Failed to run action: %s", err.Error())
		}
	},
}

// ListAutoCompleteTasks returns a list of all of the available tasks that can be run
func ListAutoCompleteTasks(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	var tasksFile types.TasksFile

	if _, err := os.Stat(config.TaskFileLocation); os.IsNotExist(err) {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	err := utils.ReadYaml(config.TaskFileLocation, &tasksFile)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	var taskNames []string
	for _, task := range tasksFile.Tasks {
		taskNames = append(taskNames, task.Name)
	}
	return taskNames, cobra.ShellCompDirectiveNoFileComp
}

func listTasksFromIncludes(rows *[][]string, tasksFile types.TasksFile) error {
	var includedTasksFile types.TasksFile

	variableConfig := runner.GetMaruVariableConfig()
	err := variableConfig.PopulateVariables(tasksFile.Variables, setRunnerVariables)
	if err != nil {
		return err
	}

	templatePattern := `\${[^}]+}`
	re := regexp.MustCompile(templatePattern)
	for _, include := range tasksFile.Includes {
		// get included TasksFile
		for includeName, includeFileLocation := range include {
			// check for templated variables in includeFileLocation value
			if re.MatchString(includeFileLocation) {
				includeFileLocation = utils.TemplateString(variableConfig.GetSetVariables(), includeFileLocation)
			}
			// check if included file is a url
			if helpers.IsURL(includeFileLocation) {
				includedTasksFile = loadTasksFromRemoteIncludes(includeFileLocation)
			} else {
				includedTasksFile = loadTasksFromLocalIncludes(includeFileLocation)
			}
			for _, task := range includedTasksFile.Tasks {
				*rows = append(*rows, []string{fmt.Sprintf("%s:%s", includeName, task.Name), task.Description})
			}
		}
	}

	return nil
}

func loadTasksFromRemoteIncludes(includeFileLocation string) types.TasksFile {
	var includedTasksFile types.TasksFile

	// Send an HTTP GET request to fetch the content of the remote file
	resp, err := http.Get(includeFileLocation)
	if err != nil {
		message.Fatalf(err, "Error fetching %s", includeFileLocation)
	}
	defer resp.Body.Close()

	// Read the content of the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		message.Fatalf(err, "Error reading contents of %s", includeFileLocation)
	}

	// Deserialize the content into the includedTasksFile
	err = goyaml.Unmarshal(body, &includedTasksFile)
	if err != nil {
		message.Fatalf(err, "Error deserializing %s into includedTasksFile", includeFileLocation)
	}
	return includedTasksFile
}

func loadTasksFromLocalIncludes(includeFileLocation string) types.TasksFile {
	var includedTasksFile types.TasksFile
	fullPath := filepath.Join(filepath.Dir(config.TaskFileLocation), includeFileLocation)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		message.Fatalf(err, "%s not found", fullPath)
	}
	err := utils.ReadYaml(fullPath, &includedTasksFile)
	if err != nil {
		message.Fatalf(err, "Failed to load file: %s", err.Error())
	}
	return includedTasksFile
}

func init() {
	initViper()
	rootCmd.AddCommand(runCmd)
	runFlags := runCmd.Flags()
	runFlags.StringVarP(&config.TaskFileLocation, "file", "f", config.TasksYAML, lang.CmdRunFlag)
	runFlags.BoolVarP(&listTasks, "list", "t", false, lang.CmdRunList)
	runFlags.BoolVarP(&listAllTasks, "list-all", "T", false, lang.CmdRunListAll)
	runFlags.BoolVar(&dryRun, "dry-run", false, lang.CmdRunDryRun)
	runFlags.StringToStringVar(&setRunnerVariables, "set", nil, lang.CmdRunSetVarFlag)
}
