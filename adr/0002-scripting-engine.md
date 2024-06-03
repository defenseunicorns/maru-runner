# 2. Scripting Engine

Date: 2024-06-03

## Status

Draft

## Context

Presently, only action types provided by `maru`[^1] are:

- `cmd:` (i.e. [`BaseAction`](https://github.com/defenseunicorns/maru-runner/blob/main/src/types/actions.go#L23)) - basic shell command execution
- `wait:` (i.e. [`ActionWait`](https://github.com/defenseunicorns/maru-runner/blob/main/src/types/actions.go#L37)), which supports two types of "status checks":
  - `cluster:` (i.e. [`ActionWaitCluster`](https://github.com/defenseunicorns/maru-runner/blob/main/src/types/actions.go#L43)) -
    perform status checks against K8s cluster resources
  - `network:` (i.e. [`ActionWaitNetwork`](https://github.com/defenseunicorns/maru-runner/blob/main/src/types/actions.go#L51)) -
    poll arbitrary HTTP/TCP endpoints for a given status code

The `wait` action is really the only abstraction that is provided around
shell scripting. As we seek to enhance or expand built-in capabilities,
we have three high-level options for doing so:

1. expand YAML-based DSL with more configuration options
2. vendor additional "tools" and encourage `./zarf <tool> [...]` pattern
3. **provide a cross-platform scripting engine (with builtins
   for common tasks)**

Below is an example of a common use case for HTTP status checks that
is not readily solved by the existing `wait.network:` probe:

```yaml
tasks:
  - description: SonarQube UI Status Check
    maxRetries: 30
    cmd: |
      STATUS=$(curl -s 'https://sonarqube.uds.dev/api/system/status' | ./uds zarf tools yq '.status')
      echo "SonarQube system status: ${STATUS}"
      if [ $STATUS != "UP" ]; then
        sleep 10
        exit 1
      fi
```

### Native Scripting Engine

[Risor](https://risor.io/) aims to be _fast and flexible scripting for
Go developers and DevOps_, making it an ideal candidate for scripting
within `maru`.

Here is the example above rewritten using [Risor syntax](https://risor.io/docs/syntax):

```yaml
- description: SonarQube UI Status Check
  maxRetries: 30
  script: |
    r := fetch('https://sonarqube.uds.dev/api/system/status').json()
    return r['status'] == 'UP'
```

Risor has a bunch of built-in modules for DevOps use cases and is totally
pluggable for our own implementation. These modules in particular come
to mind as potentially useful for delivery:

- [`aws`](https://risor.io/docs/modules/aws)
- [`vault`](https://risor.io/docs/modules/vault)
- [`kubernetes`](https://risor.io/docs/modules/kubernetes)
- [`pgx`](https://risor.io/docs/modules/pgx)

Risor can be embedded into `maru` as a library, which means that all
script execution would happen natively in the Go runtime. This has
huge advantages for both portability and security. As with the
[vendoring tools](#vendor-tools) approach, we can continue to ship a
single binary with minimal-to-zero external dependencies, but with
the additional advantage of not having to rely on a host-specific shell.

### Alternatives

#### YAML-based DSL

For the specific case on network status checks, we could add support
for `jsonpath`-based `condition` checks (and align with existing
`wait.cluster.condition`).

```yaml
- description: SonarQube UI Status Check
  maxRetries: 30
  wait:
    network:
      protocol: https
      address: sonarqube.uds.dev/api/system/status
      code: 200
      condition: '{.body.status}'='UP'
```

Getting the API for `wait.network.condition` would be challenging,
though. Do we assume the response body is always JSON? If not, what
sort of expressions would be available for HTML or text responses?

#### Vendor Tools

Note that the original example depended on `curl`. As a result, this
is not necessarily portable. It may work on WSL, but probably not
vanilla Powershell. Though it is widely available, most Linux
distributions also do not ship with `curl`.

```yaml
tasks:
  - description: SonarQube UI Status Check
    maxRetries: 30
    cmd: |
      STATUS=$(./uds zarf tools curl -s 'https://sonarqube.uds.dev/api/system/status' | ./uds zarf tools yq '.status')
      echo "SonarQube system status: ${STATUS}"
      if [ $STATUS != "UP" ]; then
        sleep 10
        exit 1
      fi
```

Speaking of Powershell, we would need to maintain lots of Unix command
rewrites for anything to be reliable. In the above example:

- `sleep 10` -> `Start-Sleep -Seconds 10`[^2]
- `curl ...` -> probably ok for the most part, but is actually aliased
  to `Invoke-WebRequest`[^3]

## Decision



## Consequences


[^1]: `wait` actions are [implemented in `zarf`](https://github.com/defenseunicorns/zarf/blob/main/src/pkg/utils/wait.go#L32) currently
[^2]: https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.utility/start-sleep
[^3]: https://stackoverflow.com/a/73956607
