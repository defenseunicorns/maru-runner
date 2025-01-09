// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package utils provides utility fns for maru
package utils

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/defenseunicorns/maru-runner/src/message"
	"github.com/defenseunicorns/pkg/helpers/v2"
	goyaml "github.com/goccy/go-yaml"
	"github.com/pterm/pterm"
)

const (
	tmpPathPrefix = "maru-"
)

// Regex to match the GitLab repo files api, test: https://regex101.com/r/mBXuyM/1
var gitlabAPIRegex = regexp.MustCompile(`\/api\/v4\/projects\/(?P<repoID>\d+)\/repository\/files\/(?P<path>[^\/]+)\/raw`)

// UseLogFile writes output to stderr and a logFile.
func UseLogFile() error {
	writer, err := message.UseLogFile("")
	logFile := writer
	if err != nil {
		return err
	}
	message.SLog.Info(fmt.Sprintf("Saving log file to %s", message.LogFileLocation()))
	logWriter := io.MultiWriter(os.Stderr, logFile)
	pterm.SetDefaultOutput(logWriter)
	return nil
}

// MergeEnv merges two environment variable arrays,
// replacing variables found in env2 with variables from env1
// otherwise appending the variable from env1 to env2
func MergeEnv(env1, env2 []string) []string {
	envMap := make(map[string]string)
	var result []string

	// First, populate the map with env2's values for quick lookup.
	for _, s := range env2 {
		split := strings.SplitN(s, "=", 2)
		if len(split) == 2 {
			envMap[split[0]] = split[1]
		}
	}

	// Then, update the map with env1's values, effectively merging them.
	for _, s := range env1 {
		split := strings.SplitN(s, "=", 2)
		if len(split) == 2 {
			envMap[split[0]] = split[1]
		}
	}

	// Finally, reconstruct the environment array from the map.
	for key, value := range envMap {
		result = append(result, key+"="+value)
	}

	return result
}

// FormatEnvVar format environment variables replacing non-alphanumeric characters with underscores and adding INPUT_ prefix
func FormatEnvVar(name, value string) string {
	// replace all non-alphanumeric characters with underscores
	name = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(name, "_")
	name = strings.ToUpper(name)
	// prefix with INPUT_ (same as GitHub Actions)
	return fmt.Sprintf("INPUT_%s=%s", name, value)
}

// ReadYaml reads a yaml file and unmarshals it into a given config.
func ReadYaml(path string, destConfig any) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot %s", err.Error())
	}

	err = goyaml.Unmarshal(file, destConfig)
	if err != nil {
		errStr := err.Error()
		lines := strings.SplitN(errStr, "\n", 2)
		return fmt.Errorf("cannot unmarshal %s: %s", path, lines[0])
	}

	return nil
}

// MakeTempDir creates a temp directory with the maru- prefix.
func MakeTempDir(basePath string) (string, error) {
	if basePath != "" {
		if err := helpers.CreateDirectory(basePath, helpers.ReadWriteExecuteUser); err != nil {
			return "", err
		}
	}

	tmp, err := os.MkdirTemp(basePath, tmpPathPrefix)
	if err != nil {
		return "", err
	}

	message.SLog.Debug(fmt.Sprintf("Using temporary directory: %s", tmp))

	return tmp, nil
}

// JoinURLPath joins a path in a URL (detecting the URL type)
func JoinURLPath(currentURLPath, includeFilePath string) (string, error) {
	get, err := helpers.MatchRegex(gitlabAPIRegex, currentURLPath)
	if err != nil {
		return path.Join(path.Dir(currentURLPath), includeFilePath), nil
	}
	escapedPath := get("path")
	repoID := get("repoID")
	unescapedPath, err := url.PathUnescape(escapedPath)
	if err != nil {
		return "", nil
	}
	joinedPath := path.Join(path.Dir(unescapedPath), includeFilePath)
	return fmt.Sprintf("/api/v4/projects/%s/repository/files/%s/raw", repoID, url.PathEscape(joinedPath)), nil
}

// ReadRemoteYaml makes a get request to retrieve a given file from a URL
func ReadRemoteYaml(location string, authentication map[string]string, destConfig any) (err error) {
	// Send an HTTP GET request to fetch the content of the remote file
	req, err := http.NewRequest("GET", location, nil)
	if err != nil {
		return fmt.Errorf("unable to initialize request for %s: %w", location, err)
	}

	parsedLocation, err := url.Parse(location)
	if err != nil {
		return fmt.Errorf("failed parsing URL %s: %w", location, err)
	}
	if token, ok := authentication[parsedLocation.Host]; ok {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to make request for %s: %w", location, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed getting %s: %s", location, resp.Status)
	}

	// Read the content of the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed reading contents of %s: %w", location, err)
	}

	// Deserialize the content into the includedTasksFile
	err = goyaml.Unmarshal(body, destConfig)
	if err != nil {
		return fmt.Errorf("failed unmarshalling contents of %s: %w", location, err)
	}

	return nil
}

func parseChecksum(src string) (string, string, error) {
	atSymbolCount := strings.Count(src, "@")
	var checksum string
	if atSymbolCount > 0 {
		parsed, err := url.Parse(src)
		if err != nil {
			return src, checksum, fmt.Errorf("unable to parse the URL: %s", src)
		}
		if atSymbolCount == 1 && parsed.User != nil {
			return src, checksum, nil
		}

		index := strings.LastIndex(src, "@")
		checksum = src[index+1:]
		src = src[:index]
	}
	return src, checksum, nil
}
