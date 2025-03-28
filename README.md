# maru-runner

[![Latest Release](https://img.shields.io/github/v/release/defenseunicorns/maru-runner)](https://github.com/defenseunicorns/maru-runner/releases)
[![Go version](https://img.shields.io/github/go-mod/go-version/defenseunicorns/maru-runner?filename=go.mod)](https://go.dev/)
[![Build Status](https://img.shields.io/github/actions/workflow/status/defenseunicorns/maru-runner/release.yaml)](https://github.com/defenseunicorns/maru-runner/actions/workflows/release.yaml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/defenseunicorns/maru-runner/badge)](https://api.securityscorecards.dev/projects/github.com/defenseunicorns/maru-runner)

Maru is a task runner that enables developers to automate builds and perform common shell tasks and shares a syntax similar to `zarf.yaml` `actions`.
Many [Zarf Actions features](https://docs.zarf.dev/ref/actions/) are also available in the runner.

## Table of Contents

- [Runner](#maru-runner)
    - [Quickstart](#quickstart)
    - [Key Concepts](#key-concepts)
        - [Tasks](#tasks)
        - [Actions](#actions)
            - [Task](#task)
            - [Cmd](#cmd)
        - [Variables](#variables)
        - [Wait](#wait)
        - [Includes](#includes)
        - [Task Inputs and Reusable Tasks](#task-inputs-and-reusable-tasks)

## Quickstart

Create a file called `tasks.yaml`

```yaml
variables:
  - name: FOO
    default: foo

tasks:
  - name: default
    actions:
      - cmd: echo "run default task"

  - name: example
    actions:
      - task: set-variable
      - task: echo-variable

  - name: set-variable
    actions:
      - cmd: echo "bar"
        setVariables:
          - name: FOO

  - name: echo-variable
    actions:
      - cmd: echo ${FOO}
```

From the same directory as the `tasks.yaml`, run the `example` task using:

```bash
run example
```

This will run the `example` tasks which in turn runs the `set-variable` and `echo-variable`. In this example, the text "
bar" should be printed to the screen twice.

Optionally, you can specify the location and name of your `tasks.yaml` using the `--file` or `-f` flag:

```bash
run example -f tmp/tasks.yaml
```

You can also view the tasks that are available to run in your current task file using the `list` flag, or you can view all tasks including tasks from external files that are being included in your task file by using the `list-all` flag:

```bash
run -f tmp/tasks.yaml --list
```
```bash
run -f tmp/tasks.yaml --list-all
```

## Key Concepts

### Tasks

Tasks are the fundamental building blocks of the runner and they define operations to be performed. The `tasks` key
at the root of `tasks.yaml` define a list of tasks to be run. This underlying operations performed by a task are defined
under the `actions` key:

```yaml
tasks:
  - name: all-the-tasks
    actions:
      - task: make-build-dir
      - task: install-deps
```

In this example, the name of the task is "all-the-tasks", and it is composed of multiple sub-tasks to run. These sub-tasks
would also be defined in the list of `tasks`:

```yaml
tasks:
  - name: default
    actions:
      - cmd: echo "run default task"

  - name: all-the-tasks
    actions:
      - task: make-build-dir
      - task: install-deps

  - name: make-build-dir
    actions:
      - cmd: mkdir -p build

  - name: install-deps
    actions:
      - cmd: go mod tidy
```

These tasks can be run individually:

```bash
run all-the-tasks   # runs all-the-tasks, which calls make-build-dir and install-deps
run make-build-dir  # only runs make-build-dir
```

#### Default Tasks
In the above example, there is also a `default` task, which is special, optional, task that can be used for the most common entrypoint for your tasks. When trying to run the `default` task, you can omit the task name from the run command:

```bash
run
```

### Actions

Actions are the underlying operations that a task will perform. Each action under the `actions` key has a unique syntax.

#### Task

A task can reference a task, thus making tasks composable.

```yaml
tasks:
  - name: foo
    actions:
      - task: bar
  - name: bar
    actions:
      - task: baz
  - name: baz
    actions:
      - cmd: "echo task foo is composed of task bar which is composed of task baz!"
```

In this example, the task `foo` calls a task called `bar` which calls a task `baz` which prints some output to the
console.

#### Cmd

Actions can run arbitrary bash commands including in-line scripts, and the output of a command can be placed in a
variable using the `setVariables` key

```yaml
tasks:
  - name: foo
    actions:
      - cmd: echo -n 'dHdvIHdlZWtzIG5vIHByb2JsZW0=' | base64 -d
        setVariables:
          - name: FOO
```

This task will decode the base64 string and set the value as a variable named `FOO` that can be used in other tasks.

Command blocks can have several other properties including:

- `description`: description of the command
    - `mute`: boolean value to mute the output of a command
    - `dir`: the directory to run the command in
    - `env`: list of environment variables to run for this `cmd` block only

      ```yaml
      tasks:
        - name: foo
          actions:
            - cmd: echo ${BAR}
              env:
                - BAR=bar
      ```

    - `maxRetries`: number of times to retry the command
    - `maxTotalSeconds`: max number of seconds the command can run until it is killed; takes precedence
      over `maxRetries`

### Variables

Variables can be defined in several ways:

1. At the top of the `tasks.yaml`

   ```yaml
   variables:
     - name: FOO
       default: foo

   tasks: ...
   ```

1. As the output of a `cmd`

   ```yaml
   variables:
     - name: FOO
       default: foo
   tasks:
     - name: foo
       actions:
         - cmd: uname -m
           mute: true
           setVariables:
             - name: FOO
         - cmd: echo ${FOO}
         # Or drop the curly brackets
         - cmd: echo $FOO
         # Or use template syntax
         - cmd: echo ${{ .variables.FOO }}
   ```

1. As an environment variable prefixed with `MARU_`. In the example above, if you create an env var `MARU_FOO=bar`, then the`FOO` variable would be set to `bar`.

1. Using the `--set` flag in the CLI : `run foo --set FOO=bar`

To use a variable, reference it using `${VAR_NAME}`

Note that variables also have the following attributes when setting them with YAML:

- `sensitive`: boolean value indicating if a variable should be visible in output
- `default`: default value of a variable
    - In the example above, if `FOO` did not have a default, and you have an environment variable `MARU_FOO=bar`, the default would get set to `bar`.

#### Environment Variable Files

To include a file containing environment variables that you'd like to load into a task, use the `envPath` key in the task. This will load all of the environment variables in the file into the task being called and its child tasks.

```yaml
tasks:
  - name: env
    actions:
      - cmd: echo $FOO
      - cmd: echo $MARU_ARCH
      - task: echo-env
  - name: echo-env
    envPath: ./path/to/.env
    actions:
      - cmd: echo different task $FOO
```

#### Automatic Environment Variables
The following Environment Variables are set automatically by maru-runner and are available to any action being performed:
- `MARU` - Set to 'true' to indicate the action was executed by maru-runner.
- `MARU_ARCH` - Set to the current architecture. e.g. 'amd64'

Example:

- tasks.yaml
  ```yaml
    - name: print-common-env
      actions:
        - cmd: echo MARU_ARCH=[$MARU_ARCH]
        - cmd: echo MARU=[$MARU]
  ```
- `maru run print-common-env` output:
  ```
      MARU_ARCH=[amd64]
    ✔  Completed "echo MARU_ARCH=[$MARU_ARCH]"
      MARU=[true]
    ✔  Completed "echo MARU=[$MARU]"
  ```

#### Variable Precedence
Variable precedence is as follows, from least to most specific:
- Variable defaults set in YAML
- Environment variables prefixed with `MARU_`
- Variables set with the `--set` flag in the CLI

That is to say, variables set via the `--set` flag take precedence over all other variables.

There are a couple of exceptions to this precedence order:
- When a variable is modified using `setVariable`, which will change the value of the variable during runtime.
- When another application is vendoring in maru, it can use config.AddExtraEnv to add extra environment variables. Any variables set by an application in this way take precedence over everything else.


### Wait

The `wait`key is used to block execution while waiting for a resource, including network responses and K8s operations

```yaml
tasks:
  - name: network-response
    wait:
      network:
        protocol: https
        address: 1.1.1.1
        code: 200
  - name: configmap-creation
    wait:
      cluster:
        kind: configmap
        name: simple-configmap
        namespace: foo
```

### Includes

The `includes` key is used to import tasks from local, remote, or OCI task files. This is useful for sharing common tasks across multiple task files. When importing a task from a local task file, the path is relative to the file you are currently in. When running a task, the tasks in the task file as well as the `includes` get processed to ensure there are no infinite loop references.

```yaml
includes:
  - local: ./path/to/tasks-to-import.yaml
  - remote: https://raw.githubusercontent.com/defenseunicorns/maru-runner/main/src/test/tasks/remote-import-tasks.yaml
  - oci-tasks: oci://ghcr.io/myorg/maru-tasks:latest

tasks:
  - name: import-local
    actions:
      - task: local:some-local-task
  - name: import-remote
    actions:
      - task: remote:echo-var
  - name: import-oci
    actions:
      - task: oci-tasks:hello-world
```

Note that included task files can also include other task files, with the following restriction:

- If a task file includes a remote task file, the included remote task file cannot include any local task files

Tasks from an included file can also be run individually, by using the includes reference name followed by a colon and the name of the task, like in the example below. Both of these commands run the same task.

```bash
run import-local
```
```bash
run local:some-local-task
```

#### OCI Task Files

Maru supports using OCI artifacts as task files. This allows you to store your tasks in container registries and version them using tags. To use an OCI task file, use the `oci://` prefix in your includes:

```yaml
includes:
  - common-tasks: oci://ghcr.io/myorg/maru-tasks:v1.0.0

tasks:
  - name: use-oci-task
    actions:
      - task: common-tasks:setup-env
```

You can also use variables in your OCI references:

```yaml
variables:
  - name: REGISTRY
    default: "ghcr.io"
  - name: TASK_VERSION
    default: "latest"

includes:
  - common-tasks: oci://${REGISTRY}/myorg/maru-tasks:${TASK_VERSION}
```

Authentication to private OCI registries works the same way as for remote HTTPS task files, using the `maru auth login` command with the registry hostname.

#### Authenticated Includes

Some included remote task files may require authentication to access - to access these you can use the `maru auth login` command to add a personal access token (bearer auth) to your computer keychain.

Below is an example of how to use the login command for the above remote:

```bash
gh auth token | maru auth login raw.githubusercontent.com --token-stdin
```

If you wish to remove a token for a given host you can run the `maru auth logout` command:

```bash
maru auth logout raw.githubusercontent.com
```

If you are running Maru on a headless system without a keyring provider you can also specify the `host:token` key-value pairs in the `MARU_AUTH` environment variable as a JSON object or in the `options.auth` section of the Maru config file:

```bash
export MARU_AUTH="{\"raw.githubusercontent.com\": \"$(gh auth token)\"}"
```

### Task Inputs and Reusable Tasks

Although all tasks should be reusable, sometimes you may want to create a task that can be reused with different inputs. To create a reusable task that requires inputs, add an `inputs` key with a map of inputs to the task:

```yaml
tasks:
  - name: echo-var
    inputs:
      hello-input:
        description: This is an input to the echo-var task
        required: true
      deprecated-input:
        default: foo
        description: this is a input from a previous version of this task
        deprecatedMessage: this input is deprecated, use hello-input instead
      input3:
        default: baz
    actions:
      # to use the input, reference it using INPUT_<INPUT_NAME> in all caps
      - cmd: echo $INPUT_HELLO_INPUT
      # or use template "index" syntax
      - cmd: echo ${{ index .inputs "hello-input" }}
      # or use simple template syntax. NOTE: This doesn't work if your input name has any dashes in it.
      - cmd: echo "${{ .inputs.input3 }}"


  - name: use-echo-var
    actions:
      - task: echo-var
        with:
          # hello-input is the name of the input in the echo-var task, hello-unicorn is the value we want to pass in
          hello-input: hello unicorn
```

In this example, the `echo-var` task takes an input called `hello-input` and prints it to the console; notice that the `input` can have a `default` value. The `use-echo-var` task calls `echo-var` with a different input value using the `with` key. In this case `"hello unicorn"` is passed to the `hello-input` input.

Note that the `deprecated-input` input has a `deprecatedMessage` attribute. This is used to indicate that the input is deprecated and should not be used. If a task is run with a deprecated input, a warning will be printed to the console.

#### Templates

When creating a task with `inputs` you can use [Go templates](https://pkg.go.dev/text/template#hdr-Functions) in that task's `actions`. For example:

```yaml
tasks:
  - name: length-of-inputs
    inputs:
      hello-input:
        default: hello world
        description: This is an input to the echo-var task
      another-input:
        default: another world
    actions:
      # index and len are go template functions, while .inputs is map representing the inputs to the task
      - cmd: echo ${{ index .inputs "hello-input" | len }}
      - cmd: echo ${{ index .inputs "another-input" | len }}

  - name: len
    actions:
      - task: length-of-inputs
        with:
          hello-input: hello unicorn
```

Running `maru run len` will print the length of the inputs to `hello-input` and `another-input` to the console.

#### Command Line Flags

> [!NOTE]
> The `--with` command line flag is experimental and likely to change as part of a comprehensive overhaul of the inputs/variables design.

When creating a task with `inputs` you can also use the `--with` command line flag. Given the `length-of-inputs` task documented above, you can also run:

```shell
maru run length-of-inputs --with hello-input="hello unicorn"
```
