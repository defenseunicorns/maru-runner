// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package cmd contains the CLI commands for maru.
package cmd

import (
	"io"
	"os"
	"strings"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/config/lang"
	"github.com/defenseunicorns/maru-runner/src/message"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

// token is the token to save for the given host
var token string

// tokenStdIn controls whether to pull the token from standard in
var tokenStdIn bool

var loginCmd = &cobra.Command{
	Use: "login host",
	PersistentPreRun: func(_ *cobra.Command, _ []string) {
		exitOnInterrupt()
		cliSetup()
	},
	Short:             lang.CmdLoginShort,
	ValidArgsFunction: ListAutoCompleteTasks,
	Args:              cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		host := args[0]

		if tokenStdIn {
			stdin, err := io.ReadAll(os.Stdin)
			if err != nil {
				message.Fatalf(err, "Unable to read the token from standard input: %s", err.Error())
			}

			token = strings.TrimSuffix(string(stdin), "\n")
			token = strings.TrimSuffix(token, "\r")
		}

		err := keyring.Set(config.KeyringService, host, token)
		if err != nil {
			message.Fatalf(err, "Unable to set the token for %s in the keyring: %s", host, err.Error())
		}
	},
}

func init() {
	initViper()
	rootCmd.AddCommand(loginCmd)
	loginFlags := loginCmd.Flags()
	loginFlags.StringVarP(&token, "token", "t", "", lang.CmdLoginTokenFlag)
	loginFlags.BoolVar(&tokenStdIn, "token-stdin", false, lang.CmdLoginTokenStdInFlag)
}
