---
name: release
description: "Prepare and tag a new semver release for kubectm. Generates changelog from commits, creates a git tag, and pushes to trigger the GitHub Actions release pipeline."
disable-model-invocation: true
---

# release

Prepare and publish a new kubectm release. This skill automates changelog generation, version tagging, and pushing to trigger the CI/CD release pipeline.

## Inputs

Parse arguments in the format: `/release <version>` where version is a semver string (e.g., `1.2.3` or `v1.2.3`).

| Parameter | Description | Required | Default |
|-----------|-------------|----------|---------|
| `version` | Semver version to release (e.g., `1.2.3`) | Yes | None |

## Steps

1. **Validate version format**: Ensure the version matches semver pattern. Prefix with `v` if not already present.

2. **Check preconditions**:
   - Confirm we are on the `main` branch
   - Confirm the working tree is clean (`git status --porcelain` is empty)
   - Confirm the tag does not already exist (`git tag -l <version>`)
   - Run `go test -race ./...` to ensure all tests pass
   - Run `go vet ./...` to check for issues

3. **Generate changelog**: Collect commits since the last tag:
   ```bash
   git log $(git describe --tags --abbrev=0 2>/dev/null || git rev-list --max-parents=0 HEAD)..HEAD --oneline --no-merges
   ```
   Group commits by type (feat, fix, chore, etc.) based on conventional commit prefixes.

4. **Show release summary**: Display the version, changelog, and ask for user confirmation before proceeding.

5. **Create and push tag**:
   ```bash
   git tag -a <version> -m "Release <version>"
   git push origin <version>
   ```

6. **Confirm**: Display a link to the GitHub Actions run that will build and publish the release.
   The Actions pipeline produces GPG signatures and build attestations automatically — wait for CI to complete rather than expecting local artefacts. The workflow generates release notes directly from the tag; no separate changelog file needs to be written locally before pushing.

## Examples

```
/release 1.0.0
/release v1.2.3
```
