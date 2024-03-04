// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

// Package cmd contains the CLI commands for maru.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/config/lang"
	"github.com/defenseunicorns/maru-runner/src/pkg/runner"
	"github.com/defenseunicorns/maru-runner/src/types"
	"github.com/defenseunicorns/zarf/src/pkg/message"
	zarfUtils "github.com/defenseunicorns/zarf/src/pkg/utils"
	"github.com/defenseunicorns/zarf/src/pkg/utils/exec"
	"github.com/defenseunicorns/zarf/src/pkg/utils/helpers"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

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
		if len(args) > 1 && !config.ListTasks {
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

		if config.ListTasks {
			rows := [][]string{
				{"Name", "Description"},
			}
			for _, task := range tasksFile.Tasks {
				rows = append(rows, []string{task.Name, task.Description})
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
	runFlags.BoolVar(&config.ListTasks, "list", false, lang.CmdRunList)
	runFlags.StringToStringVar(&config.SetRunnerVariables, "set", nil, lang.CmdRunSetVarFlag)
}
