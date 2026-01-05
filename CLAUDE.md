# CLAUDE.md

This file provides guidance for Claude Code when working on this repository.

## Project Overview

**kubectm** is a Go CLI tool that simplifies Kubernetes kubeconfig management across multiple cloud providers. It automatically retrieves credentials, downloads kubeconfig files from cloud providers, and merges them into `~/.kube/config`.

## Architecture

```
kubectm/
├── cmd/main.go              # CLI entry point, flag parsing, main workflow
├── pkg/
│   ├── credentials/         # Cloud provider credential retrieval
│   │   ├── retrieve.go      # Main credential retrieval logic
│   │   ├── aws.go           # AWS credential handling (stub)
│   │   ├── azure.go         # Azure credential handling (stub)
│   │   ├── gcp.go           # GCP credential handling (stub)
│   │   └── linode.go        # Linode credential handling (implemented)
│   ├── kubeconfig/          # Kubeconfig operations
│   │   ├── download.go      # Download kubeconfigs from providers
│   │   ├── merge.go         # Merge kubeconfigs into ~/.kube/config
│   │   ├── rename.go        # Rename clusters/contexts
│   │   └── linode.go        # Linode-specific kubeconfig handling
│   ├── ui/
│   │   └── prompt.go        # Interactive credential selection UI
│   └── utils/
│       └── logging.go       # Logging utilities and credential obfuscation
```

## Build Commands

```bash
# Build the binary
go build -o kubectm ./cmd

# Run tests with coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# View coverage
go tool cover -func=coverage.out

# Cross-compile for different platforms
GOOS=linux GOARCH=amd64 go build -o kubectm-linux-amd64 ./cmd
GOOS=darwin GOARCH=arm64 go build -o kubectm-darwin-arm64 ./cmd
GOOS=windows GOARCH=amd64 go build -o kubectm-windows-amd64.exe ./cmd
```

## Key Dependencies

- `k8s.io/client-go` - Kubernetes client library for kubeconfig handling
- `k8s.io/apimachinery` - Kubernetes API types
- `github.com/AlecAivazis/survey/v2` - Interactive prompts
- `github.com/fatih/color` - Colored terminal output

## Development Notes

### Supported Cloud Providers

- **Linode (LKE)**: Fully implemented - reads from `LINODE_API_TOKEN` env var or `linode-cli` config
- **AWS, Azure, GCP**: Stub implementations only - not yet functional

### Configuration Storage

- Selected providers saved to: `~/.kubectm/selected_providers.json`
- Credentials path: `~/.kubectm/selected_credentials.json`

### CLI Flags

- `-h, --help`: Show help message
- `-v, --version`: Show version
- `--reset-creds`: Reset stored credentials and prompt for new ones

### Code Patterns

1. **Logging**: Use the predefined loggers (`infoLogger`, `warnLogger`, `errorLogger`, `actionLogger`) with ISO 8601 timestamps
2. **Credentials**: The `Credential` struct has `Provider` (string) and `Details` (map[string]string) fields
3. **Error Handling**: Return errors rather than fatal logging in package functions; let main handle fatal errors

### Testing

Tests use Go's standard testing package. Test files are colocated with source files:
- `pkg/kubeconfig/linode_test.go`
- `pkg/kubeconfig/merge_test.go`

### Release Process

Releases are automated via GitHub Actions when a version tag (`v*`) is pushed:
1. Tests run first
2. Security scan with Snyk
3. Cross-platform builds (linux/darwin/windows × amd64/arm64)
4. GPG signature generation
5. GitHub release with attestation
