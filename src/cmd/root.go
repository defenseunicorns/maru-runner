// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package cmd contains the CLI commands for maru.
package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/config/lang"
	"github.com/defenseunicorns/maru-runner/src/pkg/utils"
	"github.com/defenseunicorns/zarf/src/pkg/message"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "maru COMMAND",
	PersistentPreRun: func(cmd *cobra.Command, _ []string) {
		exitOnInterrupt()

		// Don't add the logo to the help command
		if cmd.Parent() == nil {
			config.SkipLogFile = true
		}
		cliSetup()
	},
	Short: lang.RootCmdShort,
	Run: func(cmd *cobra.Command, _ []string) {
		_, _ = fmt.Fprintln(os.Stderr)
		err := cmd.Help()
		if err != nil {
			message.Fatal(err, "error calling help command")
		}
	},
}

// Execute is the entrypoint for the CLI.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

// RootCmd returns the root command.
func RootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	initViper()

	v.SetDefault(V_LOG_LEVEL, "info")
	v.SetDefault(V_ARCHITECTURE, "")
	v.SetDefault(V_NO_LOG_FILE, false)
	v.SetDefault(V_TMP_DIR, "")
	v.SetDefault(V_ENV_PREFIX, "RUN")

	rootCmd.PersistentFlags().StringVarP(&config.LogLevel, "log-level", "l", v.GetString(V_LOG_LEVEL), lang.RootCmdFlagLogLevel)
	rootCmd.PersistentFlags().StringVarP(&config.CLIArch, "architecture", "a", v.GetString(V_ARCHITECTURE), lang.RootCmdFlagArch)
	rootCmd.PersistentFlags().BoolVar(&message.NoProgress, "no-progress", v.GetBool(V_NO_PROGRESS), lang.RootCmdFlagNoProgress)
	rootCmd.PersistentFlags().BoolVar(&config.SkipLogFile, "no-log-file", v.GetBool(V_NO_LOG_FILE), lang.RootCmdFlagSkipLogFile)
	rootCmd.PersistentFlags().StringVar(&config.TempDirectory, "tmpdir", v.GetString(V_TMP_DIR), lang.RootCmdFlagTempDir)
}

func cliSetup() {
	match := map[string]message.LogLevel{
		"warn":  message.WarnLevel,
		"info":  message.InfoLevel,
		"debug": message.DebugLevel,
		"trace": message.TraceLevel,
	}

	printViperConfigUsed()

	// No log level set, so use the default
	if config.LogLevel != "" {
		if lvl, ok := match[config.LogLevel]; ok {
			message.SetLogLevel(lvl)
			message.Debug("Log level set to " + config.LogLevel)
		} else {
			message.Warn(lang.RootCmdErrInvalidLogLevel)
		}
	}

	if !config.SkipLogFile && !ListTasks && !ListAllTasks {
		utils.UseLogFile()
	}
}

// exitOnInterrupt catches an interrupt and exits with fatal error
func exitOnInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		message.Fatal(lang.ErrInterrupt, lang.ErrInterrupt.Error())
	}()
}
