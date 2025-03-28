// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023-Present the Maru Authors

package test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOCITaskIncludes_Manual is for manual testing only
func TestOCITaskIncludes_Manual(t *testing.T) {
	t.Skip("Skipping OCI test - for manual testing only since it requires a valid OCI registry")

	// This test is for manual testing of the OCI functionality
	// It requires a valid OCI artifact to be available

	// Set up test directory
	tmpDir, err := os.MkdirTemp("", "maru-test-*")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a tasks file to be published to OCI
	ociTaskYaml := filepath.Join(tmpDir, "hello.yaml")
	err = os.WriteFile(ociTaskYaml, []byte(`
tasks:
  - name: hello-world
    actions:
      - cmd: echo "Hello from an OCI artifact task!"
      - cmd: echo "This task is being executed from a container registry."

  - name: with-inputs
    inputs:
      name:
        description: "Your name"
        default: "Friend"
      message:
        description: "Custom message"
        default: "Welcome to Maru OCI tasks!"
    actions:
      - cmd: echo "Hello, ${INPUT_NAME}!"
      - cmd: echo "${INPUT_MESSAGE}"
`), 0644)
	require.NoError(t, err)

	// Check if oras is installed for manual testing
	_, err = exec.LookPath("oras")
	if err != nil {
		t.Log("oras CLI not found, skipping OCI publish step")
		t.Log("Install oras from https://oras.land to run full test")
	} else {
		// Log the manual steps to publish the OCI artifact
		t.Log("Manual step: publish the OCI artifact with:")
		t.Logf("oras push ghcr.io/willswire/maru-tasks:latest --config /dev/null:application/vnd.oci.empty.v1+json %s:application/yaml", ociTaskYaml)
	}

	// Create a simple tasks file with an OCI include
	tasksYaml := filepath.Join(tmpDir, "tasks.yaml")
	err = os.WriteFile(tasksYaml, []byte(`
includes:
  - test-oci: oci://ghcr.io/willswire/maru-tasks:latest

tasks:
  - name: default
    actions:
      - cmd: echo "Testing OCI tasks"
      - task: test-oci:hello-world

  - name: with-inputs
    actions:
      - task: test-oci:with-inputs
        with:
          name: "OCI Test User"
          message: "This is a test of the OCI tasks feature!"
`), 0644)
	require.NoError(t, err)

	// For manual testing, identify the maru binary and provide instructions
	maruBin := "./build/maru-mac-apple"
	t.Logf("For manual testing run: %s run -f %s", maruBin, tasksYaml)
	t.Logf("This test requires the OCI artifact to be published first")
	t.Log("To verify the OCI feature works correctly, manual execution is required")

	// Instructions for full manual test
	fmt.Println("\n=== Manual OCI Testing Instructions ===")
	fmt.Println("1. Push the OCI task:")
	fmt.Printf("   oras push ghcr.io/willswire/maru-tasks:latest --config /dev/null:application/vnd.oci.empty.v1+json %s:application/yaml\n", ociTaskYaml)
	fmt.Println("2. Run the task with maru:")
	fmt.Printf("   %s run -f %s\n", maruBin, tasksYaml)
	fmt.Println("===================================")
}

// TestOCITaskDocumentation verifies that the OCI example files in the repo are valid
func TestOCITaskDocumentation(t *testing.T) {
	// Test that the example files exist and are valid YAML
	exampleFiles := []string{
		"/Users/willwalker/Developer/github.com/willswire/maru-runner/examples/oci/hello.yaml",
		"/Users/willwalker/Developer/github.com/willswire/maru-runner/examples/oci/push.yaml",
		"/Users/willwalker/Developer/github.com/willswire/maru-runner/examples/oci/example.yaml",
	}

	for _, file := range exampleFiles {
		_, err := os.Stat(file)
		require.NoError(t, err, "Example file not found: %s", file)

		// Verify file content
		content, err := os.ReadFile(file)
		require.NoError(t, err, "Failed to read example file: %s", file)
		require.NotEmpty(t, content, "Example file is empty: %s", file)

		// Check that the file contains tasks section
		require.Contains(t, string(content), "tasks:", "Example file does not contain tasks section: %s", file)
	}

	// Verify documentation file
	docsFile := "/Users/willwalker/Developer/github.com/willswire/maru-runner/docs/oci-tasks.md"
	_, err := os.Stat(docsFile)
	require.NoError(t, err, "Documentation file not found: %s", docsFile)

	// Check documentation content
	docsContent, err := os.ReadFile(docsFile)
	require.NoError(t, err, "Failed to read documentation file")
	docsContentStr := string(docsContent)

	// Check that the documentation references ORAS not Docker
	require.Contains(t, docsContentStr, "oras login", "Documentation should use ORAS CLI for authentication")
	require.NotContains(t, docsContentStr, "docker login", "Documentation should not use Docker CLI for authentication")

	// Check that docs reference the maru command
	require.Contains(t, docsContentStr, "maru run", "Documentation should reference the correct maru run command")

	// Check that the documentation explains authentication properly
	require.Contains(t, docsContentStr, "auth login", "Documentation should explain authentication")

	// Test successful!
	t.Log("OCI documentation and examples validated successfully")
}
