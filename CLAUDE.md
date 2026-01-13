# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
# Build the binary
go build -o kubectm ./cmd

# Run tests with coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# Display coverage report
go tool cover -func=coverage.out

# Run a single test file
go test -v ./pkg/kubeconfig/merge_test.go ./pkg/kubeconfig/merge.go

# Run tests matching a pattern
go test -v -run TestMerge ./pkg/kubeconfig/...

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o kubectm-linux-amd64 ./cmd
GOOS=darwin GOARCH=arm64 go build -o kubectm-darwin-arm64 ./cmd
GOOS=windows GOARCH=amd64 go build -o kubectm-windows-amd64.exe ./cmd
```

## Architecture Overview

kubectm is a CLI tool that downloads and merges Kubernetes configurations from cloud providers into `~/.kube/config`.

### Package Structure

- **cmd/main.go** - Entry point. Handles CLI flags, loads/saves selected providers to `~/.kubectm/selected_providers.json`, orchestrates credential retrieval and kubeconfig operations.

- **pkg/credentials/** - Provider credential discovery
  - `retrieve.go` - Central dispatcher with `RetrieveAll()` and `RetrieveSelected()` functions
  - `linode.go` - Reads from `LINODE_ACCESS_TOKEN` env var or `~/.config/linode-cli` config file
  - `aws.go`, `azure.go`, `gcp.go` - Stub implementations (not yet functional)

- **pkg/kubeconfig/** - Kubeconfig operations
  - `download.go` - Dispatcher that routes to provider-specific downloaders
  - `linode.go` - Calls Linode API (`/lke/clusters` and `/lke/clusters/{id}/kubeconfig`) to fetch kubeconfigs
  - `merge.go` - Merges `.yaml` files from `~/.kube/` into the main config, handles context naming conflicts, adds Aptakube extension for Linode icon
  - `lke.png` - Embedded Linode icon (via `//go:embed`)

- **pkg/ui/prompt.go** - Interactive multi-select for credential providers using `survey/v2`

- **pkg/utils/logging.go** - Shared loggers (`InfoLogger`, `WarnLogger`, `ErrorLogger`, `ActionLogger`) with colored prefixes

### Data Flow

1. On first run: discover all available credentials → prompt user to select → save selection
2. On subsequent runs: load saved provider selection → retrieve credentials for those providers
3. For each provider: download kubeconfigs to `~/.kube/{label}-kubeconfig.yaml`
4. Merge all `.yaml` files into `~/.kube/config`, then delete the temporary files

### Linode API Integration

The Linode provider uses API v4 (`https://api.linode.com/v4`):
- `GET /lke/clusters` - List all LKE clusters
- `GET /lke/clusters/{clusterId}/kubeconfig` - Get base64-encoded kubeconfig

Authentication via Bearer token from either `LINODE_ACCESS_TOKEN` env var or `linode-cli` config.

## Key Dependencies

- `k8s.io/client-go` - Kubernetes client library for kubeconfig handling
- `k8s.io/apimachinery` - Kubernetes API types
- `github.com/AlecAivazis/survey/v2` - Interactive prompts
- `github.com/fatih/color` - Colored terminal output

## Key Patterns

- Path traversal protection: All file operations in `merge.go` validate paths are within `~/.kube/`
- Credential obfuscation: `utils.ObfuscateCredential()` masks sensitive values in logs
- Context conflict resolution: When merging, same-cluster contexts are skipped; different-cluster same-name contexts are overwritten
- Version injection: Build with `-ldflags "-X main.Version=..."` for release versioning
- Logging: Use the predefined loggers with ISO 8601 timestamps
- Error Handling: Return errors rather than fatal logging in package functions; let main handle fatal errors

## CLI Flags

- `-h, --help`: Show help message
- `-v, --version`: Show version
- `--reset-creds`: Reset stored credentials and prompt for new ones

## Release Process

Releases are automated via GitHub Actions when a version tag (`v*`) is pushed:
1. Tests run first
2. Security scan with Snyk
3. Cross-platform builds (linux/darwin/windows × amd64/arm64)
4. GPG signature generation
5. GitHub release with attestation
