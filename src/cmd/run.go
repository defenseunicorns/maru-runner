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
	"github.com/defenseunicorns/zarf/src/pkg/message"
	zarfUtils "github.com/defenseunicorns/zarf/src/pkg/utils"
	"github.com/defenseunicorns/zarf/src/pkg/utils/exec"
	"github.com/defenseunicorns/zarf/src/pkg/utils/helpers"
	goyaml "github.com/goccy/go-yaml"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// ListTasks is a flag to print available tasks in a TaskFileLocation
var ListTasks bool

// ListAllTasks is a flag to print available tasks in a TaskFileLocation
var ListAllTasks bool

var runCmd = &cobra.Command{
	Use: "run",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		exec.ExitOnInterrupt()
		cliSetup()
	},
	Short: lang.RootCmdShort,
	ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
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
	},
	Args: func(_ *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("accepts 0 or 1 arg(s), received %d", len(args))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var tasksFile types.TasksFile

		// ensure vars are uppercase
		config.SetRunnerVariables = helpers.TransformMapKeys(config.SetRunnerVariables, strings.ToUpper)

		err := zarfUtils.ReadYaml(config.TaskFileLocation, &tasksFile)
		if err != nil {
			message.Fatalf(err, "Cannot unmarshal %s", config.TaskFileLocation)
		}

		ListTasks = cmd.Flag("list").Value.String() == "true"
		ListAllTasks = cmd.Flag("list-all").Value.String() == "true"

		if ListTasks || ListAllTasks {
			rows := [][]string{
				{"Name", "Description"},
			}
			for _, task := range tasksFile.Tasks {
				rows = append(rows, []string{task.Name, task.Description})
			}
			// If ListAllTasks, add tasks from included files
			if ListAllTasks {
				var includedTasksFile types.TasksFile
				var fullPath string
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

						} else {
							fullPath = filepath.Join(filepath.Dir(config.TaskFileLocation), includeFileLocation)
							if _, err := os.Stat(fullPath); os.IsNotExist(err) {
								message.Fatalf(err, "%s not found", fullPath)
							}
							err := zarfUtils.ReadYaml(fullPath, &includedTasksFile)
							if err != nil {
								message.Fatalf(err, "Cannot unmarshal %s", fullPath)
							}
						}

						for _, task := range includedTasksFile.Tasks {
							rows = append(rows, []string{fmt.Sprintf("%s:%s", includeName, task.Name), task.Description})
						}
					}
				}
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
		if err := runner.Run(tasksFile, taskName, config.SetRunnerVariables); err != nil {
			message.Fatalf(err, "Failed to run action: %s", err)
		}
	},
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
