# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
# Build the binary
go build -o kubectm ./cmd

# Vet the code
go vet ./...

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
  - `retrieve.go` - Central dispatcher with `RetrieveAll()` and `RetrieveSelected()` functions; includes `logCredentialDiscovery()` helper for obfuscated logging
  - `linode.go` - Reads from `LINODE_ACCESS_TOKEN` env var or `~/.config/linode-cli` config file
  - `aws.go` - Reads from `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` env vars or `~/.aws/credentials` file
  - `gcp.go` - Reads `GOOGLE_APPLICATION_CREDENTIALS` env var or gcloud ADC file; project from `GOOGLE_CLOUD_PROJECT`, credentials JSON, or gcloud active config
  - `azure.go` - Reads `AZURE_CLIENT_ID`/`AZURE_CLIENT_SECRET`/`AZURE_TENANT_ID` env vars; subscription from `AZURE_SUBSCRIPTION_ID` or `~/.azure/azureProfile.json`

- **pkg/kubeconfig/** - Kubeconfig operations
  - `download.go` - Dispatcher that routes to provider-specific downloaders (Linode, AWS)
  - `linode.go` - Calls Linode API (`/lke/clusters` and `/lke/clusters/{id}/kubeconfig`) to fetch kubeconfigs
  - `aws.go` - Downloads EKS kubeconfigs: auto-discovers regions via EC2 DescribeRegions, lists/describes EKS clusters in parallel, generates exec-based kubeconfigs using `aws eks get-token`
  - `gcp.go` - Downloads GKE kubeconfigs via the GKE REST API; OAuth tokens obtained directly (service-account JWT grant or authorized-user refresh grant); generates exec-based kubeconfigs using `gke-gcloud-auth-plugin`
  - `azure.go` - Downloads AKS kubeconfigs via Azure Resource Manager (client-credentials token, `managedClusters` list, `listClusterUserCredential`)
  - `dryrun.go` - `--dry-run` support: lists clusters per provider and reports what a merge would change without writing files
  - `merge.go` - Merges `.yaml` files from `~/.kube/` into the main config, handles context naming conflicts, adds Aptakube extension for Linode icon
  - `backup.go` - Backs up `~/.kube/config` to `~/.kube/config.bak.{timestamp}` before merge; prunes old backups (keeps last N, default 5)
  - `rename.go` - Stub for renaming clusters and contexts in kubeconfig files
  - `lke.png` - Embedded Linode icon (via `//go:embed`)

- **pkg/ui/prompt.go** - Interactive multi-select for credential providers using `survey/v2`

- **pkg/utils/logging.go** - Shared loggers (`InfoLogger`, `WarnLogger`, `ErrorLogger`, `ActionLogger`) with colored prefixes

### Data Flow

1. On first run: discover all available credentials → prompt user to select → save selection to `~/.kubectm/selected_providers.json`
2. On subsequent runs: load saved provider selection → retrieve credentials for those providers
3. For each provider: download kubeconfigs to `~/.kube/{label}-kubeconfig.yaml`
4. Back up the existing `~/.kube/config` to `~/.kube/config.bak.{timestamp}` (keeping the last N backups)
5. Merge all `.yaml` files into `~/.kube/config`, then delete the temporary files

### Linode API Integration

The Linode provider uses API v4 (`https://api.linode.com/v4`):
- `GET /lke/clusters` - List all LKE clusters
- `GET /lke/clusters/{clusterId}/kubeconfig` - Get base64-encoded kubeconfig

Authentication via Bearer token from either `LINODE_ACCESS_TOKEN` env var or `linode-cli` config.

### AWS EKS Integration

The AWS provider uses AWS SDK v2:
- `EC2 DescribeRegions` - Auto-discover enabled regions (with optional `~/.kubectm/config.json` override)
- `EKS ListClusters` - List cluster names per region (with pagination)
- `EKS DescribeCluster` - Get cluster endpoint + CA certificate

Regions are scanned in parallel (concurrency=5, 30s timeout). Generated kubeconfigs use the `aws eks get-token` exec plugin for authentication. Context naming: `{cluster-name}@{region}`.

Authentication via static credentials from the discovered `Credential.Details` (AccessKey, SecretKey, optional SessionToken).

### GCP GKE Integration

The GCP provider uses the GKE REST API (`https://container.googleapis.com/v1`):
- `GET /projects/{projectID}/locations/-/clusters` - List clusters across all locations

OAuth2 access tokens are obtained without SDK dependencies: service account keys use the signed-JWT bearer grant against the key's `token_uri`; gcloud application default credentials use the refresh-token grant. Generated kubeconfigs use the `gke-gcloud-auth-plugin` exec plugin. Context naming: `{cluster-name}@{location}`.

### Azure AKS Integration

The Azure provider uses Azure Resource Manager (`https://management.azure.com`):
- `GET /subscriptions/{sub}/providers/Microsoft.ContainerService/managedClusters` - List AKS clusters (with `nextLink` pagination)
- `POST {clusterId}/listClusterUserCredential` - Get the cluster's kubeconfig (base64, written directly like Linode)

Authentication via the OAuth2 client-credentials grant against Microsoft Entra ID (`https://login.microsoftonline.com`). Context naming: `{cluster-name}@{resource-group}`.

## Key Dependencies

- `k8s.io/client-go` - Kubernetes client library for kubeconfig handling
- `k8s.io/apimachinery` - Kubernetes API types
- `github.com/aws/aws-sdk-go-v2` - AWS SDK for EC2/EKS API calls
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
- `--backup-count <n>`: Number of kubeconfig backups to keep (default: 5)
- `--dry-run`: List available clusters and show what would change without modifying any files

## Release Process

Releases are automated via GitHub Actions when a version tag (`v*`) is pushed:
1. Tests run first
2. Cross-platform builds (linux/darwin/windows × amd64/arm64)
3. GPG signature generation
4. GitHub release with attestation
