# GitHub Actions Workflows

This directory contains the CI/CD workflows for kubectm.

## Active Workflows

### 1. `build.yml` - Build, Test & Release
**Triggers:**
- Push to `main` branch (runs tests only)
- Pull requests to `main` (runs tests)
- Push of tags matching `v*` (runs full build, test, and release)
- Manual dispatch

**Jobs:**
1. **test** - Runs Go tests with race detection and coverage reporting
2. **prepare-build** - Extracts version and prepares build metadata (tag pushes only)
3. **build** - Builds binaries for multiple platforms (tag pushes only):
   - Linux (amd64, arm64)
   - macOS/Darwin (amd64, arm64)
   - Windows (amd64, arm64)
4. **release** - Creates GitHub release with:
   - Auto-generated release notes
   - Signed binaries
   - SHA256 checksums
   - Build attestations

**Creating a Release:**
```bash
# Create and push a tag
git tag v1.0.0
git push origin v1.0.0

# The workflow will automatically:
# 1. Run tests
# 2. Build binaries for all platforms
# 3. Sign binaries with GPG
# 4. Create a GitHub release
# 5. Upload all assets
```

## Deprecated Workflows

### `release.yml.deprecated`
This workflow has been consolidated into `build.yml`. Release creation is now automatic when version tags are pushed. The file is retained for historical reference only.

## Secrets Required

The following secrets must be configured in the repository settings:

- `GPG_PRIVATE_KEY` - GPG private key for signing binaries
- `GPG_PASSPHRASE` - Passphrase for the GPG private key
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

## Action Versions

All actions use pinned versions or SHA commits for security:
- `actions/checkout@v4`
- `actions/setup-go@v5`
- `actions/upload-artifact@v4`
- `actions/download-artifact@v4`
- `softprops/action-gh-release@v2`
- `actions/attest-build-provenance@v1`

## Workflow Improvements (2025)

Recent updates include:
- ✅ Consolidated build and release into single workflow
- ✅ Added automated test execution
- ✅ Added automated release creation on tag push
- ✅ Added coverage reporting
- ✅ Added SHA256 checksums to releases
- ✅ Improved error handling and conditional execution
- ✅ Added Windows executable extension handling (.exe)

## Development Workflow

### For Contributors
1. Create a branch
2. Make changes
3. Push and create a PR to `main`
4. Workflows will run tests
5. Merge when checks pass

### For Maintainers
1. Merge approved PRs to `main`
2. When ready for release:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```
3. Workflow automatically creates release with binaries

## Troubleshooting

**Tests failing?**
- Run locally: `go test -v -race ./...`
- Check coverage: `go test -coverprofile=coverage.out ./...`

**Build failing?**
- Verify Go version matches `go.mod`
- Check for platform-specific build issues
- Review build logs in Actions tab

**Release not created?**
- Ensure tag starts with `v` (e.g., `v1.0.0`)
- Check if all jobs passed
- Verify GitHub token permissions
