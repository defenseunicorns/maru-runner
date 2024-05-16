// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

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
	"github.com/defenseunicorns/maru-runner/src/pkg/runner"
	"github.com/defenseunicorns/maru-runner/src/pkg/utils"
	"github.com/defenseunicorns/maru-runner/src/types"
	"github.com/defenseunicorns/pkg/helpers"
	zarfCommon "github.com/defenseunicorns/zarf/src/cmd/common"
	"github.com/defenseunicorns/zarf/src/pkg/message"
	zarfUtils "github.com/defenseunicorns/zarf/src/pkg/utils"
	goyaml "github.com/goccy/go-yaml"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// ListTasks is a flag to print available tasks in a TaskFileLocation (no includes)
var ListTasks bool

// ListAllTasks is a flag to print available tasks in a TaskFileLocation
var ListAllTasks bool

var runCmd = &cobra.Command{
	Use: "run",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		zarfCommon.ExitOnInterrupt()
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

		// ensure vars are uppercase
		config.SetRunnerVariables = helpers.TransformMapKeys(config.SetRunnerVariables, strings.ToUpper)

		err := zarfUtils.ReadYaml(config.TaskFileLocation, &tasksFile)
		if err != nil {
			message.Fatalf(err, "Cannot unmarshal %s", config.TaskFileLocation)
		}

		if ListTasks || ListAllTasks {
			rows := [][]string{
				{"Name", "Description"},
			}
			for _, task := range tasksFile.Tasks {
				rows = append(rows, []string{task.Name, task.Description})
			}
			// If ListAllTasks, add tasks from included files
			if ListAllTasks {
				listTasksFromIncludes(&rows, tasksFile)
			}

			err := pterm.DefaultTable.WithHasHeader().WithData(rows).Render()
			if err != nil {
				message.Fatal(err, "error listing tasks")
			}

			return
		}

		taskName := "default"
		if len(args) > 0 {
			taskName = args[0]
		}
		if err := runner.Run(tasksFile, taskName, config.SetRunnerVariables, config.GetExtraEnv()); err != nil {
			message.Fatalf(err, "Failed to run action: %s", err)
		}
	},
}

// ListAutoCompleteTasks returns a list of all of the available tasks that can be run
func ListAutoCompleteTasks(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	var tasksFile types.TasksFile

	if _, err := os.Stat(config.TaskFileLocation); os.IsNotExist(err) {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	err := zarfUtils.ReadYaml(config.TaskFileLocation, &tasksFile)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	var taskNames []string
	for _, task := range tasksFile.Tasks {
		taskNames = append(taskNames, task.Name)
	}
	return taskNames, cobra.ShellCompDirectiveNoFileComp
}

func listTasksFromIncludes(rows *[][]string, tasksFile types.TasksFile) {
	var includedTasksFile types.TasksFile
	templatePattern := `\${[^}]+}`
	re := regexp.MustCompile(templatePattern)
	for _, include := range tasksFile.Includes {
		// get included TasksFile
		for includeName, includeFileLocation := range include {
			// check for templated variables in includeFileLocation value
			if re.MatchString(includeFileLocation) {
				templateMap := utils.PopulateTemplateMap(tasksFile.Variables, config.SetRunnerVariables)
				includeFileLocation = utils.TemplateString(templateMap, includeFileLocation)
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
	err := zarfUtils.ReadYaml(fullPath, &includedTasksFile)
	if err != nil {
		message.Fatalf(err, "Cannot unmarshal %s", fullPath)
	}
	return includedTasksFile
}

func init() {
	initViper()
	rootCmd.AddCommand(runCmd)
	runFlags := runCmd.Flags()
	runFlags.StringVarP(&config.TaskFileLocation, "file", "f", config.TasksYAML, lang.CmdRunFlag)
	runFlags.BoolVar(&ListTasks, "list", false, lang.CmdRunList)
	runFlags.BoolVar(&ListAllTasks, "list-all", false, lang.CmdRunListAll)
	runFlags.StringToStringVar(&config.SetRunnerVariables, "set", nil, lang.CmdRunSetVarFlag)
}
