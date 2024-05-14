# maru-runner

[![Latest Release](https://img.shields.io/github/v/release/defenseunicorns/maru-runner)](https://github.com/defenseunicorns/maru-runner/releases)
[![Go version](https://img.shields.io/github/go-mod/go-version/defenseunicorns/maru-runner?filename=go.mod)](https://go.dev/)
[![Build Status](https://img.shields.io/github/actions/workflow/status/defenseunicorns/maru-runner/release.yaml)](https://github.com/defenseunicorns/maru-runner/actions/workflows/release.yaml)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/defenseunicorns/maru-runner/badge)](https://api.securityscorecards.dev/projects/github.com/defenseunicorns/maru-runner)

Maru is a task runner that enables developers to automate builds and perform common shell tasks. It
uses [Zarf](https://zarf.dev/) under the hood to perform tasks and shares a syntax similar to `zarf.yaml` manifests.
Many [Zarf Actions features](https://docs.zarf.dev/ref/actions/) are also available in
the runner.

## Table of Contents

- [Runner](#maru-runner)
    - [Quickstart](#quickstart)
    - [Key Concepts](#key-concepts)
        - [Tasks](#tasks)
        - [Actions](#actions)
            - [Task](#task)
            - [Cmd](#cmd)
        - [Variables](#variables)
        - [Files](#files)
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
    - `maxTotalSeconds`: max number of seconds the command can run until it is killed; takes precendence
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

That is to say, variables set via the `--set` flag take precedence over all other variables. The exception to this precedence order is when a variable is modified using `setVariable`, which will change the value of the variable during runtime.

### Files

The `files` key is used to copy local or remote files to the current working directory

```yaml
tasks:
  - name: copy-local
    files:
      - source: /tmp/foo
        target: foo
  - name: copy-remote
    files:
      - source: https://cataas.com/cat
        target: cat.jpeg
```

Files blocks can also use the following attributes:

- `executable`: boolean value indicating if the file is executable
- `shasum`: SHA string to verify the integrity of the file
- `symlinks`: list of strings referring to symlink the file to

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

The `includes` key is used to import tasks from either local or remote task files. This is useful for sharing common tasks across multiple task files. When importing a task from a local task file, the path is relative to the file you are currently in. When running a task, the tasks in the task file as well as the `includes` get processed to ensure there are no infinite loop references.

```yaml
includes:
  - local: ./path/to/tasks-to-import.yaml
  - remote: https://raw.githubusercontent.com/defenseunicorns/maru-runner/main/src/test/tasks/remote-import-tasks.yaml

tasks:
  - name: import-local
    actions:
      - task: local:some-local-task
  - name: import-remote
    actions:
      - task: remote:echo-var
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

### Task Inputs and Reusable Tasks

Although all tasks should be reusable, sometimes you may want to create a task that can be reused with different inputs. To create a reusable task that requires inputs, add an `inputs` key with a map of inputs to the task:

```yaml
tasks:
  - name: echo-var
    inputs:
      hello-input:
        default: hello world
        description: This is an input to the echo-var task
      deprecated-input:
        default: foo
        description: this is a input from a previous version of this task
        deprecatedMessage: this input is deprecated, use hello-input instead
    actions:
      # to use the input, reference it using INPUT_<INPUT_NAME> in all caps
      - cmd: echo $INPUT_HELLO_INPUT

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

Running `run len` will print the length of the inputs to `hello-input` and `another-input` to the console.
