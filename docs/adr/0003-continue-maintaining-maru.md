# 3. Continue Maintaining Maru

Date: 2026-11-03

## Status

Accepted

Supersedes [ADR 0002: Redesign of Task Variables and/or Inputs](0002-redesign-of-task-variables-and-or-inputs.md).

## Context

The product team placed Maru into maintenance mode after ADR 0002 proposed a net-new redesign of the task variable and input systems, potentially including a full rewrite. That redesign did not proceed. In practice, treating it as the expected destination discouraged incremental improvements without producing a replacement.

Maru remains part of UDS CLI and continues to need bug fixes, dependency and security updates, releases, compatibility work, and improvements driven by users. A community delivery team has formed to restore active stewardship of the project and make Maru "undead."

Restoring Maru does not settle its long-term place in the UDS product portfolio. UDS CLI may continue to vendor Maru, or the ongoing product discussion may select another task runner. The delivery team can provide implementation experience, user feedback, constraints, prototypes, and recommendations to inform that decision, but it does not own the product decision by maintaining Maru.

## Decision

Maru will return to active maintenance and incremental development. The delivery team will make Maru undead and sustain it as the task runner vendored into UDS CLI until a decision is made to continue with Maru or vendor another solution.

This work includes:

- fixing bugs and security issues;
- updating dependencies and maintaining releases;
- maintaining a supported integration with UDS CLI;
- improving documentation and contributor experience; and
- accepting well-scoped features that improve current user workflows and can be supported by the maintainers.

ADR 0002's net-new redesign is no longer the committed direction. Changes to variables, inputs, or related interfaces may proceed incrementally when they provide clear value and preserve an appropriate compatibility path. A full rewrite is not required to improve Maru.

The delivery team will help inform the longer-term direction, but its maintenance does not include choosing a replacement for Maru or indefinitely developing speculative replacement architecture. If the product process selects another solution for UDS CLI, that decision must define the migration, compatibility, deprecation, and ownership plan before active Maru maintenance ends.

## Consequences

- Maru users and contributors have a clear signal that fixes, updates, and well-scoped features are welcome.
- UDS CLI retains a supported task runner while its longer-term direction remains open.
- Maintainers can improve the current design without waiting for a rewrite or broader product decision.
- The delivery team must balance new work against compatibility and ongoing maintenance capacity.
- Existing design debt will be addressed incrementally as priorities and maintenance capacity allow.
- Maintaining Maru does not guarantee that it will remain in UDS CLI permanently.
- Replacing Maru in UDS CLI will require an explicit transition plan.
