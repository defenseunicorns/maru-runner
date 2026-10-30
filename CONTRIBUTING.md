# Contributing to Maru
Maru is actively maintained, and contributions for new features, bug fixes, documentation, dependency updates, and security updates are welcome.

Welcome :unicorn: to Maru! If you'd like to contribute, please reach out to one of the [CODEOWNERS](CODEOWNERS) and we'll be happy to get you started!

Below are some notes on our core software design philosophies that should help guide contributors.

## Code Quality and Standards
Fundamentally, software engineering is a communication problem; we write code for each other, not a computer. When working on this project (or any project!) keep your fellow humans in mind and write clearly and concisely. Below are some general guidelines for code quality and standards that make Maru :sparkles:

- **Write tests that give confidence**: Unless there is a technical blocker, every new feature and bug fix should be tested in the project's automated test suite. Although many of our tests are E2E, unit and integration-style tests are also welcomed. Note that unit tests can live in a `*_test.go` file alongside the source code, and E2E tests live in `src/test/e2e`

- **Prefer readability over being clever**: We have a strong preference for code readability in Maru. Specifically, this means things like: naming variables appropriately, keeping functions to a reasonable size and avoiding complicated solutions when simple ones exist.

- **User experience is paramount**: Maru doesn't have a pretty UI (yet), but the core user-centered design principles that apply when building a frontend also apply to this CLI tool. First and foremost, features in Maru should enhance workflows and make life easier for end users; if a feature doesn't accomplish this, it will be dropped.

### Pre-Commit Hooks and Linting
In this repo you can optionally use [pre-commit](https://pre-commit.com/) hooks for automated validation and linting, but if not CI will run these checks for you.

## Continuous Delivery
Continuous Delivery is core to our development philosophy. Check out [https://minimumcd.org](https://minimumcd.org/) for a good baseline agreement on what that means.

Specifically:

- We do trunk-based development (`main`) with short-lived feature branches that originate from the trunk, get merged into the trunk, and are deleted after the merge
- We don't merge code into `main` that isn't releasable
- We perform automated testing on all changes before they get merged to `main`
- We create immutable release artifacts

## Release Process

Maintainers release Maru by pushing a version tag to GitHub. The [release workflow](.github/workflows/release.yaml) runs on tags matching `v*`; [.goreleaser.yaml](.goreleaser.yaml) defines the published artifacts and Homebrew updates.

### Prepare a Release

1. Ensure the changes to release have merged into `main` and their CI checks have passed.
2. Choose an unused version tag in the form `vMAJOR.MINOR.PATCH`. For a release candidate, use `vMAJOR.MINOR.PATCH-rc1` (incrementing the candidate number as needed). Use `rc` for release candidates because the workflow selects the Homebrew formula by checking whether the tag contains `rc`.
3. Review the changes since the previous release so you can verify the generated release notes after publication.
4. Ensure you have permission to push release tags. The publishing job uses the GitHub `release` environment; satisfy any approval requirements configured for that environment. The workflow also requires `MARU_GITLAB_TOKEN` for E2E tests and `HOMEBREW_TAP_WORKFLOW_GITHUB_APP_ID` and `HOMEBREW_TAP_WORKFLOW_GITHUB_APP_SECRET` for Homebrew tap access.

### Create and Push the Tag

From a clean working tree, fetch the latest `main` and tags, review the commit to release, then create an annotated tag on that commit. Replace the example version below with the chosen version:

```sh
git fetch origin main --tags
git log -1 --oneline origin/main
git tag -a vX.Y.Z origin/main -m "Release vX.Y.Z"
git push origin vX.Y.Z
```

Pushing the tag starts **Release Maru on Tag** in GitHub Actions. Creating a GitHub release manually is not required.

### Monitor Publication

1. Monitor the [release workflow run](https://github.com/defenseunicorns/maru-runner/actions/workflows/release.yaml). It builds the Linux AMD64 CLI, runs unit and E2E tests, then runs GoReleaser after validation succeeds and any environment approvals are satisfied.
2. Verify the published [GitHub release](https://github.com/defenseunicorns/maru-runner/releases). GoReleaser generates GitHub release notes and publishes Linux and macOS binaries for AMD64 and ARM64, a universal macOS binary, `checksums.txt` with SHA-256 checksums, and binary SBOMs. Releases are published directly, and prerelease status is determined automatically from the version tag.
3. Review and merge the generated pull request in [defenseunicorns/homebrew-tap](https://github.com/defenseunicorns/homebrew-tap). Stable tags update `maru`; tags containing `rc` update `maru-rc`. Both also create a versioned formula, `maru@<version>`, without the leading `v`.
4. Download a release binary, verify it against `checksums.txt`, and confirm it runs and reports the expected version with `maru version`.

If a workflow job fails, inspect its logs and any partially published artifacts or Homebrew changes before retrying. Keep published tags and artifacts immutable; if a published release needs a code fix, merge the fix and release a new version.
