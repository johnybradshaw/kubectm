# GitHub Actions Workflows

This directory contains the CI/CD workflows for kubectm.

## Active Workflows

### 1. `build.yml` - Build, Test & Release
**Triggers:**
- Push to `main` branch (runs tests only)
- Pull requests to `main` (runs tests and security scan)
- Push of tags matching `v*` (runs full build, test, scan, and release)
- Manual dispatch

**Jobs:**
1. **test** - Runs Go tests with race detection and coverage reporting
2. **scan** - Performs Snyk security scanning (Code and Open Source)
3. **prepare-build** - Extracts version and prepares build metadata (tag pushes only)
4. **build** - Builds binaries for multiple platforms (tag pushes only):
   - Linux (amd64, arm64)
   - macOS/Darwin (amd64, arm64)
   - Windows (amd64, arm64)
5. **release** - Creates GitHub release with:
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
# 2. Run security scans
# 3. Build binaries for all platforms
# 4. Sign binaries with GPG
# 5. Create a GitHub release
# 6. Upload all assets
```

### 2. `snyk-security.yml` - Snyk Security Scan
**Triggers:**
- Push to `main` branch
- Pull requests to `main`
- Daily at 2 AM UTC (scheduled)
- Manual dispatch

**Jobs:**
1. **snyk** - Runs Snyk vulnerability scanning:
   - Go dependency scanning
   - Uploads results to GitHub Code Scanning

**Note:** This workflow runs independently of the build workflow to provide continuous security monitoring, including scheduled daily scans.

## Deprecated Workflows

### `release.yml.deprecated`
This workflow has been consolidated into `build.yml`. Release creation is now automatic when version tags are pushed. The file is retained for historical reference only.

## Secrets Required

The following secrets must be configured in the repository settings:

- `SNYK_TOKEN` - Snyk API token for security scanning
- `GPG_PRIVATE_KEY` - GPG private key for signing binaries
- `GPG_PASSPHRASE` - Passphrase for the GPG private key
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

## Action Versions

All actions use pinned versions or SHA commits for security:
- `actions/checkout@v4`
- `actions/setup-go@v5`
- `actions/upload-artifact@v4`
- `actions/download-artifact@v4`
- `snyk/actions/setup@1d730b57b6e0f430d884a538caa90d6d93e0bf15` (v0.4.0)
- `snyk/actions/golang@1d730b57b6e0f430d884a538caa90d6d93e0bf15` (v0.4.0)
- `github/codeql-action/upload-sarif@v3`
- `softprops/action-gh-release@v2`
- `actions/attest-build-provenance@v1`

## Workflow Improvements (2025)

Recent updates include:
- ✅ Consolidated build and release into single workflow
- ✅ Added automated test execution
- ✅ Updated deprecated Snyk action references (@master → pinned SHA)
- ✅ Added automated release creation on tag push
- ✅ Added coverage reporting
- ✅ Added SHA256 checksums to releases
- ✅ Added scheduled security scans
- ✅ Improved error handling and conditional execution
- ✅ Added Windows executable extension handling (.exe)

## Development Workflow

### For Contributors
1. Create a branch
2. Make changes
3. Push and create a PR to `main`
4. Workflows will run tests and security scans
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

**Security scan failing?**
- Check Snyk dashboard for vulnerability details
- Update dependencies: `go get -u ./...`
- Review and fix reported issues

**Build failing?**
- Verify Go version matches `go.mod`
- Check for platform-specific build issues
- Review build logs in Actions tab

**Release not created?**
- Ensure tag starts with `v` (e.g., `v1.0.0`)
- Check if all jobs passed
- Verify GitHub token permissions
