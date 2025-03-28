# Using OCI Artifacts for Maru Tasks

This document explains how to publish and use Maru tasks as OCI artifacts.

## What are OCI Artifacts?

OCI (Open Container Initiative) artifacts are content stored in container registries that follow the OCI distribution spec. Unlike traditional container images, OCI artifacts can be any type of content, including YAML files.

## Benefits of Using OCI for Maru Tasks

- **Versioning**: Use tags to version your task files
- **Access Control**: Leverage registry permissions for access control
- **Distribution**: Use existing container registry infrastructure
- **Immutability**: Once pushed, artifacts can be made immutable
- **Metadata**: Add annotations and labels to describe your tasks

## Publishing Tasks to an OCI Registry

To publish your Maru tasks to an OCI registry, you'll need the `oras` CLI tool.

### 1. Install ORAS

```bash
# Install via Homebrew on macOS
brew install oras

# Or download a release from https://github.com/oras-project/oras/releases
```

### 2. Prepare Your Tasks File

Create a regular Maru tasks file:

```yaml
# hello.yaml
tasks:
  - name: world
    actions:
      - cmd: echo "Hello from an OCI artifact task!"
      - cmd: echo "This task is being executed from a container registry."

  - name: inputs
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
```

### 3. Push to an OCI Registry

You can push your tasks file manually:

```bash
# Log in to your registry
# For GitHub Container Registry
echo $GITHUB_TOKEN | oras login ghcr.io -u USERNAME --password-stdin

# For other registries
oras login registry.example.com

# Push the file as an OCI artifact
oras push ghcr.io/myorg/maru-tasks/hello:0.0.1 \
  --config /dev/null:application/vnd.oci.empty.v1+json \
  hello.yaml:application/yaml
```

Or use the provided `push.yaml` tasks file to automate the process:

```yaml
# push.yaml
variables:
  - name: REF
    default: ghcr.io/willswire/maru-tasks/hello:0.0.1
    description: the full OCI artifact reference (registry/repo:tag)
  - name: TASK
    default: hello.yaml
    description: the task file to publish as an OCI artifact

tasks:
  - name: default
    description: publish a task file as an OCI artifact
    actions:
      - description: check dependencies and validate variables
        cmd: |
          # Check for the oras CLI
          if ! command -v oras &> /dev/null; then
            echo "oras CLI not found. Please install it from https://oras.land"
            exit 1
          fi

          # Validate REF variable
          if [[ -z "$REF" ]]; then
            echo "ERROR: REF variable is not set"
            exit 1
          fi

          # Validate TASK variable and check if file exists
          if [[ -z "$TASK" ]]; then
            echo "ERROR: TASK variable is not set"
            exit 1
          fi

          if [[ ! -f "$TASK" ]]; then
            echo "ERROR: Task file '$TASK' not found"
            exit 1
          fi
      - description: push the artifact
        cmd: |
          oras push "$REF" \
            --config /dev/null:application/vnd.oci.empty.v1+json \
            $TASK:application/yaml
```

You can run it with:

```bash
maru run -f push.yaml
# Or with custom variables
maru run -f push.yaml --set REF=ghcr.io/myorg/maru-tasks/hello:1.0.0 --set TASK=my-tasks.yaml
```

## Using OCI Tasks in Maru

Once published, you can include the OCI tasks in your Maru task files:

```yaml
# example.yaml
includes:
  - hello: oci://ghcr.io/willswire/maru-tasks/hello:0.0.1

tasks:
  - name: default
    actions:
      - cmd: echo "Testing OCI artifact integration"
      - task: hello:world

  - name: with-args
    actions:
      - task: hello:inputs
        with:
          name: "OCI Test User"
          message: "This is a test of the OCI tasks feature!"
```

You can also use variables in your OCI references:

```yaml
variables:
  - name: REGISTRY
    default: "ghcr.io"
  - name: ORG
    default: "willswire"
  - name: REPO
    default: "maru-tasks"
  - name: TAG
    default: "latest"

includes:
  # Include tasks from OCI registry
  - oci-tasks: oci://${REGISTRY}/${ORG}/${REPO}:${TAG}

tasks:
  - name: default
    actions:
      - cmd: echo "Using tasks from OCI registry ${REGISTRY}/${ORG}/${REPO}:${TAG}"
      - task: oci-tasks:hello-world

  - name: with-custom-input
    actions:
      - task: oci-tasks:with-inputs
        with:
          name: "OCI User"
          message: "OCI tasks make sharing reusable tasks easy!"
```

### Authentication

To authenticate with private registries, use the `maru auth login` command:

```bash
# For GitHub packages
gh auth token | maru auth login ghcr.io --token-stdin

# For other registries
maru auth login registry.example.com
```

## Example: Creating a Reusable Task Library

Create a task library with common utilities:

```yaml
# ci-tasks.yaml
tasks:
  - name: lint
    actions:
      - cmd: golangci-lint run

  - name: test
    actions:
      - cmd: go test ./...

  - name: build
    inputs:
      output-dir:
        description: "Directory to output built binaries"
        default: "./build"
    actions:
      - cmd: go build -o ${INPUT_OUTPUT_DIR}/app main.go
```

Push it to your registry:

```bash
oras push ghcr.io/myorg/ci-tasks:v1.0.0 \
  --config /dev/null:application/vnd.oci.empty.v1+json \
  ci-tasks.yaml:application/yaml
```

Use it in your projects:

```yaml
# tasks.yaml in your project
includes:
  - ci: oci://ghcr.io/myorg/ci-tasks:v1.0.0

tasks:
  - name: default
    actions:
      - task: ci:lint
      - task: ci:test
      - task: ci:build
        with:
          output-dir: "./dist"
```

## Validating OCI Tasks Integration

You can validate that your OCI tasks integration is working correctly by running the following:

1. First, publish a task file to an OCI registry:

```bash
# Create a simple hello world task file
cat << EOF > hello.yaml
tasks:
  - name: hello-world
    actions:
      - cmd: echo "Hello from an OCI artifact task!"

  - name: with-inputs
    inputs:
      name:
        description: "Your name"
        default: "Friend"
    actions:
      - cmd: echo "Hello, \${INPUT_NAME}!"
EOF

# Login to your registry
oras login ghcr.io -u <username>

# Push the task file
oras push ghcr.io/myorg/maru-tasks:latest \
  --config /dev/null:application/vnd.oci.empty.v1+json \
  hello.yaml:application/yaml
```

2. Create a local task file that includes the OCI tasks:

```bash
cat << EOF > local-tasks.yaml
includes:
  - remote: oci://ghcr.io/myorg/maru-tasks:latest

tasks:
  - name: default
    actions:
      - cmd: echo "Testing OCI tasks integration"
      - task: remote:hello-world
EOF
```

3. Run the task with Maru:

```bash
maru run -f local-tasks.yaml
```

If everything is set up correctly, you should see output from both your local task and the task loaded from the OCI registry.

## Best Practices

1. **Version Your Tasks**: Use semantic versioning in your tags
2. **Publish to Organizations**: Prefer org-level repositories over personal ones
3. **Document Inputs**: Include documentation for any task inputs
4. **Keep Tasks Focused**: Create separate OCI artifacts for different concerns
5. **Consider Immutability**: Use immutable tags for production use
6. **Use Variables**: Parameterize registry, repository, and tag information in your OCI references
7. **Test Before Publishing**: Validate your tasks locally before pushing to a registry
8. **Include Examples**: Add example usage in your task documentation
