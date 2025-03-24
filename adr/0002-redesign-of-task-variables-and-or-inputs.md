# 2. Redesign of Task Variables and/or Inputs

Date: 2025-03-24

## Status

Accepted

## Context

Maru Runner currently supports two mechanisms for task configuration: Variables and Inputs. This duality has led to overlapping functionality, inconsistencies in user experience, and increased maintenance overhead. The competing solutions have created ambiguity for both new and existing users regarding which configuration method to use. The goal of this redesign is to simplify the system by removing redundant approaches, ensuring consistency, and reducing technical debt.

### Variables

- The Variables pattern was copied over from Zarf.
- They are defined at the root tasks file level, rather than per task.
- Variables do not have feature parity with Inputs. They currently lack:
    - `required:` field, which allows the task file maintainer to require a non-empty value
    - `deprecatedMessage:` field, which allows the task file maintainer to set an input as deprecated, which will alert the consumer via a command line message if they use it.
    - The ability to use yaml to set variables, similar to Inputs' `with:` yaml syntax.

### Inputs

- The Inputs pattern was designed and implemented as part of a Dash Days project.
- They are defined per task.
- Inputs do not have feature parity with Variables. They currently lack:
    - `pattern:` field, which allows the task file maintainer to define a regex in order to validate the input value
    - `prompt:` field, which allows the task file maintainer to tell Maru to prompt the consumer interactively for the value if one isn't already defined, rather than making the task fail by returning an error message and a nonzero exit code.
    - The ability to set an input using an environment variable, similar to Variables' `MARU_` env var prefix.
    - The ability to set the value of an input from a previous task action, similar to Variables' `setVariable:` yaml syntax.

## Decision

TBD

## Consequences

The following outlines the pros and cons of the various solutions considered:

### Option 1: Achieve feature parity for Inputs, then deprecate Variables

**Pros:**
- **Unified Configuration:** Consolidates functionality into a single system (Inputs), simplifying both the user interface and internal codebase.
- **Maintenance Efficiency:** Reduces redundant code paths and minimizes future maintenance by focusing on one mechanism.
- **Clear Migration Path:** Provides a straightforward deprecation plan for Variables, encouraging users to transition gradually.

**Cons:**
- **Migration Challenges:** Existing users reliant on Variables may face disruption or require additional migration tooling.
- **Implementation Overhead:** Bringing Inputs to full parity with Variables, and adding deprecation notices to Variables, will demand development, testing, and documentation efforts.
- **Backward Compatibility Risks:** There is a risk of breaking changes that could affect current workflows during the transition period.

### Option 2: Achieve feature parity for Variables, then deprecate Inputs

**Pros:**
- **Unified Configuration:** Consolidates functionality into a single system (Variables), simplifying both the user interface and internal codebase.
- **Maintenance Efficiency:** Reduces redundant code paths and minimizes future maintenance by focusing on one mechanism.
- **Clear Migration Path:** Provides a straightforward deprecation plan for Inputs, encouraging users to transition gradually.

**Cons:**
- **Migration Challenges:** Existing users reliant on Inputs may face disruption or require additional migration tooling.
- **Implementation Overhead:** Bringing Variables to full parity with Inputs, and adding deprecation notices to Inputs, will demand development, testing, and documentation efforts.
- **Backward Compatibility Risks:** There is a risk of breaking changes that could affect current workflows during the transition period.


### Option 3: Maintain both patterns with improved documentation and clear use-cases

**Pros:**
- **Minimal Disruption:** Keeping both Variables and Inputs avoids immediate breaking changes, preserving current workflows.
- **Flexibility:** Users can choose the system that best suits their needs, which may be beneficial for diverse use cases.
- **Short-term Stability:** Requires fewer code changes, which can be attractive if resources or time are limited.

**Cons:**
- **Continued Confusion:** Having two overlapping yet distinct patterns may perpetuate user uncertainty and inconsistent usage patterns.
- **Increased Maintenance:** Maintaining parallel solutions leads to duplicated efforts, complicating bug fixes and feature updates.
- **Technical Debt:** The coexistence of both systems may hinder long-term scalability and the implementation of future improvements.

### Option 4: Come up with something net-new, then deprecate both Variables and Inputs

> [!NOTE]
> This option could result in changes to the existing Maru tool. It could also potentially involve a full rewrite, leading to a `maru2`. More investigation is necessary if this option is chosen as the Decision.

**Pros:**
- **Modern Approach:** A new solution can be designed from the ground up with modern requirements in mind, free from legacy constraints.
- **Unified and Simplified System:** Offers the opportunity to create a cohesive, intuitive configuration system that eliminates past inconsistencies.
- **Lessons Learned:** Can incorporate lessons from the shortcomings of both Variables and Inputs, potentially resulting in a more robust, flexible, and scalable solution.
- **Enhanced Developer Experience:** A well-designed, new API may streamline the configuration process for both users and developers.

**Cons:**
- **High Development Cost:** Requires significant initial investment in design, development, testing, and documentation.
- **Risk of Unproven System:** Introducing an entirely new configuration system carries risks of unforeseen issues and bugs.
- **Transition Complexity:** Migrating from two existing systems to a new solution may necessitate comprehensive migration tooling and detailed user guidance.
- **Potential Instability:** The transition period may be challenging, with potential disruptions as users adapt to the new system.

Each option carries trade-offs between user experience, development effort, and long-term maintainability. Further discussion and evaluation of project priorities are necessary before finalizing the chosen path forward.
