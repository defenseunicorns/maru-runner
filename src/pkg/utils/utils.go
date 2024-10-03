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
	"path/filepath"
	"regexp"
	"strings"

	"github.com/defenseunicorns/maru-runner/src/config/lang"
	"github.com/defenseunicorns/maru-runner/src/message"
	"github.com/defenseunicorns/pkg/helpers/v2"
	goyaml "github.com/goccy/go-yaml"
	"github.com/pterm/pterm"
)

const (
	tmpPathPrefix = "maru-"
)

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

// DownloadToFile downloads a given URL to the target filepath
func DownloadToFile(src string, dst string) (err error) {
	message.SLog.Debug(fmt.Sprintf("Downloading %s to %s", src, dst))
	// check if the parsed URL has a checksum
	// if so, remove it and use the checksum to validate the file
	src, checksum, err := parseChecksum(src)
	if err != nil {
		return err
	}

	err = helpers.CreateDirectory(filepath.Dir(dst), helpers.ReadWriteExecuteUser)
	if err != nil {
		return fmt.Errorf(lang.ErrCreatingDir, filepath.Dir(dst), err.Error())
	}

	// Create the file
	file, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf(lang.ErrWritingFile, dst, err.Error())
	}
	defer file.Close()

	err = httpGetFile(src, file)
	if err != nil {
		return err
	}

	// If the file has a checksum, validate it
	if len(checksum) > 0 {
		received, err := helpers.GetSHA256OfFile(dst)
		if err != nil {
			return err
		}
		if received != checksum {
			return fmt.Errorf("shasum mismatch for file %s: expected %s, got %s ", dst, checksum, received)
		}
	}

	return nil
}

func httpGetFile(url string, destinationFile *os.File) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("unable to download the file %s", url)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad HTTP status: %s", resp.Status)
	}

	// Writer the body to file
	title := fmt.Sprintf("Downloading %s", filepath.Base(url))
	progressBar := message.NewProgressBar(resp.ContentLength, title)

	if _, err = io.Copy(destinationFile, io.TeeReader(resp.Body, progressBar)); err != nil {
		message.SLog.Debug(err.Error())
		progressBar.Failf("Unable to save the file %s", destinationFile.Name())
		return err
	}

	title = fmt.Sprintf("Downloaded %s", url)
	progressBar.Successf("%s", title)

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
