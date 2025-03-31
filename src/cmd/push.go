// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

// Package cmd contains the CLI commands for maru.
package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/defenseunicorns/maru-runner/src/config"
	"github.com/defenseunicorns/maru-runner/src/message"
	v1 "github.com/opencontainers/image-spec/specs-go/v1" // ocispec
	"github.com/spf13/cobra"
	keyring "github.com/zalando/go-keyring"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// OCI artifact media types
const (
	yamlMediaType       = "application/yaml"
	emptyConfigType     = "application/vnd.oci.empty.v1+json"
	defaultYamlArtifact = "application/vnd.oci.image.manifest.v1+json"
)

var pushCmd = &cobra.Command{
	Use:   "push TASK_FILE OCI_REFERENCE",
	Short: "Push a task file to an OCI registry",
	Long: `Push a Maru task file to an OCI registry.

Examples:
  # Push a task file to GitHub Container Registry
  maru push tasks.yaml ghcr.io/myorg/maru-tasks:latest

  # Push a task file with a specific tag
  maru push tasks.yaml ghcr.io/myorg/maru-tasks:v1.0.0
`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskFile := args[0]
		reference := args[1]

		return pushTaskFile(taskFile, reference)
	},
}

func init() {
	initViper()
	rootCmd.AddCommand(pushCmd)
}

// Parse an OCI reference into registry, repository, and tag
func parseOCIReference(reference string) (registry string, repository string, tag string, err error) {
	// Verify reference starts with oci:// prefix
	if !strings.HasPrefix(reference, "oci://") {
		return "", "", "", fmt.Errorf("reference must start with 'oci://', got: %s", reference)
	}

	// Remove oci:// prefix
	reference = strings.TrimPrefix(reference, "oci://")

	// Format expected: registry/repo/path:tag
	parts := strings.SplitN(reference, "/", 2)
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid reference format: %s", reference)
	}

	registry = parts[0]   // e.g., ghcr.io
	remainder := parts[1] // e.g., myorg/maru-tasks:0.0.1

	// Split the remainder at the colon to get repo path and tag
	repoAndTag := strings.SplitN(remainder, ":", 2)
	repository = repoAndTag[0] // e.g., myorg/maru-tasks
	tag = "latest"
	if len(repoAndTag) > 1 {
		tag = repoAndTag[1] // e.g., 0.0.1
	}

	return registry, repository, tag, nil
}

// Push a task file to an OCI registry
func pushTaskFile(taskFilePath, reference string) error {
	ctx := context.Background()

	// Verify the task file exists
	if _, err := os.Stat(taskFilePath); os.IsNotExist(err) {
		return fmt.Errorf("task file not found: %s", taskFilePath)
	}

	// Create a temporary directory for the file store
	tmpDir, err := os.MkdirTemp("", "maru-push-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file store
	fs, err := file.New(tmpDir)
	if err != nil {
		return fmt.Errorf("failed to create file store: %w", err)
	}
	defer fs.Close()

	// Parse the OCI reference
	registry, repoPath, tag, err := parseOCIReference(reference)
	if err != nil {
		return err
	}

	// Full repository reference
	fullRepo := fmt.Sprintf("%s/%s", registry, repoPath)
	message.SLog.Info(fmt.Sprintf("Pushing %s to %s:%s", taskFilePath, fullRepo, tag))

	// Get absolute path of the task file (for reading)
	taskFileAbs, err := filepath.Abs(taskFilePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	// Use only the base name to avoid absolute path issues in the artifact.
	taskFileName := filepath.Base(taskFilePath)

	// Create the empty config file (empty content to mimic /dev/null)
	configFileName := "config.json"
	configFileFull := filepath.Join(tmpDir, configFileName)
	if err := os.WriteFile(configFileFull, []byte(""), 0644); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	// Add the empty config to the file store using a relative name.
	configDesc, err := fs.Add(ctx, configFileName, emptyConfigType, configFileFull)
	if err != nil {
		return fmt.Errorf("failed to add config file: %w", err)
	}

	// Add the task file to the file store with a relative name.
	taskDesc, err := fs.Add(ctx, taskFileName, yamlMediaType, taskFileAbs)
	if err != nil {
		return fmt.Errorf("failed to add task file: %w", err)
	}

	// Pack the files into a manifest
	layers := []v1.Descriptor{taskDesc}
	manifestDesc, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1, defaultYamlArtifact, oras.PackManifestOptions{
		Layers:           layers,
		ConfigDescriptor: &configDesc,
	})
	if err != nil {
		return fmt.Errorf("failed to pack manifest: %w", err)
	}

	// Tag the manifest
	err = fs.Tag(ctx, manifestDesc, tag)
	if err != nil {
		return fmt.Errorf("failed to tag manifest: %w", err)
	}

	// Create a new repository client
	repo, err := remote.NewRepository(fullRepo)
	if err != nil {
		return fmt.Errorf("failed to create repository client: %w", err)
	}

	// Try to get token from keyring for the registry
	token, err := keyring.Get(config.KeyringService, registry)
	if err == nil && token != "" {
		// Configure authentication
		authClient := &auth.Client{
			Client: http.DefaultClient,
			Cache:  auth.NewCache(),
			Credential: auth.StaticCredential(registry, auth.Credential{
				Username: "token",
				Password: token,
			}),
		}
		repo.Client = authClient
	} else {
		message.SLog.Debug(fmt.Sprintf("No authentication token found for %s", registry))
		message.SLog.Info(fmt.Sprintf("You may need to authenticate using 'maru auth login %s --token <YOUR_TOKEN>'", registry))
	}

	// Copy from the file store to the remote repository
	_, err = oras.Copy(ctx, fs, tag, repo, tag, oras.DefaultCopyOptions)
	if err != nil {
		return fmt.Errorf("failed to push OCI artifact: %w", err)
	}

	message.SLog.Info(fmt.Sprintf("Successfully pushed %s to %s", taskFilePath, reference))
	return nil
}
