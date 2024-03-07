// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present The UDS Authors

// Package runner provides functions for running tasks in a tasks.yaml
package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "unsafe"
	// used for compile time directives to pull functions from Zarf
	_ "github.com/defenseunicorns/zarf/src/pkg/packager" // import for the side effect of bringing in actions fns

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/pkg/utils"
	"github.com/defenseunicorns/maru-runner/src/types"
	"github.com/defenseunicorns/zarf/src/pkg/message"
	zarfUtils "github.com/defenseunicorns/zarf/src/pkg/utils"
	zarfTypes "github.com/defenseunicorns/zarf/src/types"
)

func (r *Runner) performAction(action types.Action) error {
	if action.TaskReference != "" {
		// todo: much of this logic is duplicated in Run, consider refactoring
		referencedTask, err := r.getTask(action.TaskReference)
		if err != nil {
			return err
		}

		// template the withs with variables
		for k, v := range action.With {
			action.With[k] = utils.TemplateString(r.TemplateMap, v)
		}

		referencedTask.Actions, err = utils.TemplateTaskActionsWithInputs(referencedTask, action.With)
		if err != nil {
			return err
		}

		withEnv := []string{}
		for name := range action.With {
			withEnv = append(withEnv, utils.FormatEnvVar(name, action.With[name]))
		}
		if err := validateActionableTaskCall(referencedTask.Name, referencedTask.Inputs, action.With); err != nil {
			return err
		}
		for _, a := range referencedTask.Actions {
			a.Env = utils.MergeEnv(withEnv, a.Env)
		}
		if err := r.executeTask(referencedTask); err != nil {
			return err
		}
	} else {
		err := r.performZarfAction(action.ZarfComponentAction)
		if err != nil {
			return err
		}
	}
	return nil
}

// processAction checks if action needs to be processed for a given task
func (r *Runner) processAction(task types.Task, action types.Action) bool {

	taskReferenceName := strings.Split(task.Name, ":")[0]
	actionReferenceName := strings.Split(action.TaskReference, ":")[0]
	// don't need to process if the action.TaskReference is empty or if the task and action references are the same since
	// that indicates the task and task in the action are in the same file
	if action.TaskReference != "" && (taskReferenceName != actionReferenceName) {
		for _, task := range r.TasksFile.Tasks {
			// check if TasksFile.Tasks already includes tasks with given reference name, which indicates that the
			// reference has already been processed.
			if strings.Contains(task.Name, taskReferenceName+":") || strings.Contains(task.Name, actionReferenceName+":") {
				return false
			}
		}
		return true
	}
	return false
}

func getUniqueTaskActions(actions []types.Action) []types.Action {
	uniqueMap := make(map[string]bool)
	var uniqueArray []types.Action

	for _, action := range actions {
		if !uniqueMap[action.TaskReference] {
			uniqueMap[action.TaskReference] = true
			uniqueArray = append(uniqueArray, action)
		}
	}
	return uniqueArray
}

func (r *Runner) performZarfAction(action *zarfTypes.ZarfComponentAction) error {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		cmdEscaped string
		out        string
		err        error

		cmd = action.Cmd
	)

	// If the action is a wait, convert it to a command.
	if action.Wait != nil {
		// If the wait has no timeout, set a default of 5 minutes.
		if action.MaxTotalSeconds == nil {
			fiveMin := 300
			action.MaxTotalSeconds = &fiveMin
		}

		// Convert the wait to a command.
		if cmd, err = convertWaitToCmd(*action.Wait, action.MaxTotalSeconds); err != nil {
			return err
		}

		// Mute the output because it will be noisy.
		t := true
		action.Mute = &t

		// Set the max retries to 0.
		z := 0
		action.MaxRetries = &z

		// Not used for wait actions.
		d := ""
		action.Dir = &d
		action.Env = []string{}
		action.SetVariables = []zarfTypes.ZarfComponentActionSetVariable{}
	}

	// load the contents of the env file into the Action + the RUN_ARCH
	if r.envFilePath != "" {
		envFilePath := filepath.Join(filepath.Dir(config.TaskFileLocation), r.envFilePath)
		envFileContents, err := os.ReadFile(envFilePath)
		if err != nil {
			return err
		}
		action.Env = append(action.Env, strings.Split(string(envFileContents), "\n")...)
	}

	// load an env var for the architecture
	action.Env = append(action.Env, fmt.Sprintf("%s_ARCH=%s", strings.ToUpper(config.EnvPrefix), config.GetArch()))

	if action.Description != "" {
		cmdEscaped = action.Description
	} else {
		cmdEscaped = message.Truncate(cmd, 60, false)
	}

	spinner := message.NewProgressSpinner("Running \"%s\"", cmdEscaped)
	// Persist the spinner output so it doesn't get overwritten by the command output.
	spinner.EnablePreserveWrites()

	cfg := actionGetCfg(zarfTypes.ZarfComponentActionDefaults{}, *action, r.TemplateMap)

	if cmd, err = actionCmdMutation(cmd); err != nil {
		spinner.Errorf(err, "Error mutating command: %s", cmdEscaped)
	}

	// Template dir string
	cfg.Dir = utils.TemplateString(r.TemplateMap, cfg.Dir)

	// template cmd string
	cmd = utils.TemplateString(r.TemplateMap, cmd)

	duration := time.Duration(cfg.MaxTotalSeconds) * time.Second
	timeout := time.After(duration)

	// Keep trying until the max retries is reached.
	for remaining := cfg.MaxRetries + 1; remaining > 0; remaining-- {

		// Perform the action run.
		tryCmd := func(ctx context.Context) error {
			// Try running the command and continue the retry loop if it fails.
			if out, err = actionRun(ctx, cfg, cmd, cfg.Shell, spinner); err != nil {
				return err
			}

			out = strings.TrimSpace(out)

			// If an output variable is defined, set it.
			for _, v := range action.SetVariables {
				// include ${...} syntax in template map for uniformity and to satisfy zarfUtils.ReplaceTextTemplate
				nameInTemplatemap := "${" + v.Name + "}"
				r.TemplateMap[nameInTemplatemap] = &zarfUtils.TextTemplate{
					Sensitive:  v.Sensitive,
					AutoIndent: v.AutoIndent,
					Type:       v.Type,
					Value:      out,
				}
				if regexp.MustCompile(v.Pattern).MatchString(r.TemplateMap[nameInTemplatemap].Value); err != nil {
					message.WarnErr(err, err.Error())
					return err
				}
			}

			// If the action has a wait, change the spinner message to reflect that on success.
			if action.Wait != nil {
				spinner.Successf("Wait for \"%s\" succeeded", cmdEscaped)
			} else {
				spinner.Successf("Completed \"%s\"", cmdEscaped)
			}

			// If the command ran successfully, continue to the next action.
			return nil
		}

		// If no timeout is set, run the command and return or continue retrying.
		if cfg.MaxTotalSeconds < 1 {
			spinner.Updatef("Waiting for \"%s\" (no timeout)", cmdEscaped)
			if err := tryCmd(context.TODO()); err != nil {
				continue
			}

			return nil
		}

		// Run the command on repeat until success or timeout.
		spinner.Updatef("Waiting for \"%s\" (timeout: %ds)", cmdEscaped, cfg.MaxTotalSeconds)
		select {
		// On timeout break the loop to abort.
		case <-timeout:
			break

		// Otherwise, try running the command.
		default:
			ctx, cancel = context.WithTimeout(context.Background(), duration)
			defer cancel()
			if err := tryCmd(ctx); err != nil {
				continue
			}

			return nil
		}
	}

	select {
	case <-timeout:
		// If we reached this point, the timeout was reached.
		return fmt.Errorf("command \"%s\" timed out after %d seconds", cmdEscaped, cfg.MaxTotalSeconds)

	default:
		// If we reached this point, the retry limit was reached.
		return fmt.Errorf("command \"%s\" failed after %d retries", cmdEscaped, cfg.MaxRetries)
	}
}

// Perform some basic string mutations to make commands more useful.
func actionCmdMutation(cmd string) (string, error) {
	runCmd, err := zarfUtils.GetFinalExecutablePath()
	if err != nil {
		return cmd, err
	}

	// Try to patch the binary path in case the name isn't exactly "./run".
	prefix := "./run"
	if config.CmdPrefix != "" {
		prefix = fmt.Sprintf("./%s", config.CmdPrefix)
	}
	cmd = strings.ReplaceAll(cmd, prefix, runCmd+" ")

	return cmd, nil
}

// convertWaitToCmd will return the wait command if it exists, otherwise it will return the original command.
func convertWaitToCmd(wait zarfTypes.ZarfComponentActionWait, timeout *int) (string, error) {
	// Build the timeout string.
	timeoutString := fmt.Sprintf("--timeout %ds", *timeout)

	// If the action has a wait, build a cmd from that instead.
	cluster := wait.Cluster
	if cluster != nil {
		ns := cluster.Namespace
		if ns != "" {
			ns = fmt.Sprintf("-n %s", ns)
		}

		// Build a call to the zarf wait-for command (uses system Zarf)
		cmd := fmt.Sprintf("zarf tools wait-for %s %s %s %s %s",
			cluster.Kind, cluster.Identifier, cluster.Condition, ns, timeoutString)

		// config.CmdPrefix is set when vendoring both the runner and Zarf
		if config.CmdPrefix != "" {
			cmd = fmt.Sprintf("./%s %s", config.CmdPrefix, cmd)
		}
		return cmd, nil
	}

	network := wait.Network
	if network != nil {
		// Make sure the protocol is lower case.
		network.Protocol = strings.ToLower(network.Protocol)

		// If the protocol is http and no code is set, default to 200.
		if strings.HasPrefix(network.Protocol, "http") && network.Code == 0 {
			network.Code = 200
		}

		// Build a call to the zarf wait-for command (uses system Zarf)
		cmd := fmt.Sprintf("zarf tools wait-for %s %s %d %s",
			network.Protocol, network.Address, network.Code, timeoutString)

		// config.CmdPrefix is set when vendoring both the runner and Zarf
		if config.CmdPrefix != "" {
			cmd = fmt.Sprintf("./%s %s", config.CmdPrefix, cmd)
		}
		return cmd, nil
	}

	return "", fmt.Errorf("wait action is missing a cluster or network")
}

// validateActionableTaskCall validates a tasks "withs" and inputs
func validateActionableTaskCall(inputTaskName string, inputs map[string]types.InputParameter, withs map[string]string) error {
	missing := []string{}
	for inputKey, input := range inputs {
		// skip inputs that are not required or have a default value
		if !input.Required || input.Default != "" {
			continue
		}
		checked := false
		for withKey, withVal := range withs {
			// verify that the input is in the with map and the "with" has a value
			if inputKey == withKey && withVal != "" {
				checked = true
				break
			}
		}
		if !checked {
			missing = append(missing, inputKey)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("task %s is missing required inputs: %s", inputTaskName, strings.Join(missing, ", "))
	}
	for withKey := range withs {
		matched := false
		for inputKey, input := range inputs {
			if withKey == inputKey {
				if input.DeprecatedMessage != "" {
					message.Warnf("This input has been marked deprecated: %s", input.DeprecatedMessage)
				}
				matched = true
				break
			}
		}
		if !matched {
			message.Warnf("Task %s does not have an input named %s", inputTaskName, withKey)
		}
	}
	return nil
}

//go:linkname actionGetCfg github.com/defenseunicorns/zarf/src/pkg/packager.actionGetCfg
func actionGetCfg(cfg zarfTypes.ZarfComponentActionDefaults, a zarfTypes.ZarfComponentAction, vars map[string]*zarfUtils.TextTemplate) zarfTypes.ZarfComponentActionDefaults

//go:linkname actionRun github.com/defenseunicorns/zarf/src/pkg/packager.actionRun
func actionRun(ctx context.Context, cfg zarfTypes.ZarfComponentActionDefaults, cmd string, shellPref zarfTypes.ZarfComponentActionShell, spinner *message.Spinner) (string, error)
