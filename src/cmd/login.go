// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package cmd contains the CLI commands for maru.
package cmd

import (
	"fmt"
	"log"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/config/lang"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var loginCmd = &cobra.Command{
	Use: "login",
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
		service := "my-app"
		user := "anon"
		password := "secret"

		// set password
		err := keyring.Set(service, user, password)
		if err != nil {
			log.Fatal(err)
		}

		// get password
		secret, err := keyring.Get(service, user)
		if err != nil {
			log.Fatal(err)
		}

		log.Println(secret)
	},
}

func init() {
	initViper()
	rootCmd.AddCommand(loginCmd)
	runFlags := runCmd.Flags()
	runFlags.StringVarP(&config.TaskFileLocation, "file", "f", config.TasksYAML, lang.CmdRunFlag)
	runFlags.BoolVar(&dryRun, "dry-run", false, lang.CmdRunDryRun)
}
