# 1. Runner Migration

Date: 2024-03-01

## Status

Accepted

## Context

Due to frustration with current build tooling (ie. Makefiles) and the need for a more custom build system for UDS, we decided to experiment with new a feature in UDS CLI called UDS Runner. This feature allowed users to define and run complex build workflows in a simple, declarative way, based on existing functionality in Zarf Actions.

### Migration

After quickly gaining adoption across the organization, we originally decided to make the UDS Runner a first-class citizen of UDS CLI. However, in an effort to reduce the scope of UDS CLI and experiment with the runner as a new standalone project, the UDS Runner functionality will be migrated to this repo. See original ADR in the UDS CLI repo [here](https://github.com/defenseunicorns/uds-cli/blob/main/adr/0002-runner.md).

### Alternatives

#### Make
Aside from concerns around syntax, maintainability and personal preference, we wanted a tool that we could use in all environments (dev, CI, prod, etc), and could support creating and deploying Zarf packages and UDS bundles. After becoming frustrated with several overly-large and complex Makefiles to perform these tasks, the team decided to explore additional tooling outside of Make.

#### Task
According to the [official docs](https://taskfile.dev/) "Task is a task runner / build tool that aims to be simpler and easier to use than, for example, GNU Make." This project was evaluated during a company Dash Days event and was found to be a good fit for our needs. However, due to the context of the larger UDS ecosystem, we are largely unable to bring in projects that have primarily non-US contributors.

** It is important to note that although the runner takes ideas from [Task](https://taskfile.dev/), Github Actions and Gitlab pipelines, it is not a direct copy of any of these tools, and implementing a particular pattern from one of these tools does not mean that all features from that tool should be implemented.

## Decision
The runner will be a standalone project called `maru` that lives in its own repo and can be vendored by other :unicorn: products.

## Consequences

The UDS CLI team will own this product for a short time before eventually handing it off to another team. Furthermore, the UDS CLI team will vendor the runner such that no breaking changes will be introduced for UDS CLI users.
