# kubectm — Product Requirements Document

**Version:** 1.0
**Date:** 2026-03-20
**Author:** Generated from project analysis
**Status:** Draft

---

## 1. Problem Statement

Engineers working across multiple cloud providers must manually download, merge, and maintain Kubernetes kubeconfig files from each provider. This involves:

- Logging into each provider's CLI or console
- Running provider-specific commands to fetch cluster configs
- Manually merging configs into `~/.kube/config` without overwriting existing contexts
- Renaming auto-generated context names (e.g., `lke12345-ctx`) to something human-readable
- Repeating this process whenever clusters are added, removed, or credentials rotate

This is tedious, error-prone, and scales poorly as the number of clusters and providers grows.

## 2. Product Vision

**kubectm** is a single CLI command that discovers your cloud credentials, downloads all Kubernetes cluster configs, and merges them into a unified `~/.kube/config` — ready to use with `kubectl`, `kubectx`, and tools like [Aptakube](https://aptakube.com).

Run `kubectm` once and your kubeconfig is current across every provider.

## 3. Current State

### What Works

| Component | Status | Notes |
|-----------|--------|-------|
| Linode credential discovery | Done | Env var (`LINODE_ACCESS_TOKEN`) + `linode-cli` config file |
| Linode kubeconfig download | Done | API v4: list clusters, fetch base64 kubeconfig per cluster |
| Kubeconfig merge | Done | Merges `~/.kube/*.yaml` into `~/.kube/config`, handles context conflicts |
| Aptakube extension | Done | Adds Linode icon to contexts for Aptakube integration |
| AWS credential discovery | Done | Env vars + `~/.aws/credentials` file, profile support |
| Interactive provider selection | Done | Multi-select prompt, selection persisted to `~/.kubectm/selected_credentials.json` |
| CLI flags | Done | `--help`, `--version`, `--reset-creds` |
| Cross-platform builds | Done | Linux/macOS/Windows x amd64/arm64, GPG signed, attested |
| Path traversal protection | Done | File operations validated within `~/.kube/` |
| Credential obfuscation | Done | Sensitive values masked in log output |

### What's Stubbed or Missing

| Component | Status | Notes |
|-----------|--------|-------|
| AWS kubeconfig download | Stub | Credentials discovered but no EKS API integration |
| Azure credential discovery | Stub | Returns `nil, nil` |
| Azure kubeconfig download | Not started | — |
| GCP credential discovery | Stub | Returns `nil, nil` |
| GCP kubeconfig download | Not started | — |
| Context/cluster renaming | Stub | `RenameConfigs()` logs and returns nil |
| Dry-run mode | Not started | No way to preview without modifying config |
| Backup before merge | Not started | No backup of existing `~/.kube/config` |

## 4. Target Providers

### 4.1 Linode / Akamai Connected Cloud (Done)

**Credential sources:**
- `LINODE_ACCESS_TOKEN` environment variable
- `~/.config/linode-cli` config file (INI format, `[DEFAULT]` section, `token` key)

**API:** Linode API v4
- `GET /lke/clusters` — list all LKE clusters
- `GET /lke/clusters/{id}/kubeconfig` — base64-encoded kubeconfig

**Context naming:** `{cluster-label}` (from cluster metadata)

### 4.2 AWS / Amazon EKS (Credential discovery done, download needed)

**Credential sources:**
- `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` / `AWS_SESSION_TOKEN` environment variables
- `~/.aws/credentials` file (INI format, profile support via `AWS_PROFILE`)

**API:** AWS EKS API
- `eks:ListClusters` — list cluster names per region
- `eks:DescribeCluster` — get cluster endpoint + CA cert
- Generate kubeconfig using `aws-iam-authenticator` or `aws eks get-token` exec plugin

**Multi-region:** Must enumerate clusters across all enabled regions (or use a configurable region list).

**Context naming:** `{cluster-name}@{region}` (e.g., `prod-cluster@us-east-1`)

**Dependencies:** AWS SDK for Go v2 (`github.com/aws/aws-sdk-go-v2`)

### 4.3 Google Cloud / GKE (Not started)

**Credential sources:**
- `GOOGLE_APPLICATION_CREDENTIALS` environment variable (service account JSON key path)
- `gcloud` CLI application default credentials (`~/.config/gcloud/application_default_credentials.json`)
- `gcloud` CLI config for active project (`~/.config/gcloud/properties`, `core/project`)

**API:** GKE API
- `container.projects.locations.clusters.list` — list clusters per project/location
- Cluster response includes `endpoint` + `masterAuth.clusterCaCertificate`

**Multi-project:** Should support discovering clusters across multiple projects if credentials allow.

**Context naming:** `{cluster-name}@{location}` (e.g., `prod-gke@us-central1`)

**Dependencies:** Google Cloud Go SDK (`cloud.google.com/go/container`)

### 4.4 Microsoft Azure / AKS (Not started)

**Credential sources:**
- `AZURE_CLIENT_ID` / `AZURE_CLIENT_SECRET` / `AZURE_TENANT_ID` / `AZURE_SUBSCRIPTION_ID` environment variables
- Azure CLI login state (`az login`) — `~/.azure/` directory
- Managed identity (when running inside Azure)

**API:** Azure Resource Manager
- List AKS clusters across resource groups in a subscription
- `GET /subscriptions/{sub}/providers/Microsoft.ContainerService/managedClusters`
- Get cluster admin/user credentials: returns kubeconfig content

**Multi-subscription:** Should support multiple subscriptions if credentials allow.

**Context naming:** `{cluster-name}@{resource-group}` (e.g., `aks-prod@rg-production`)

**Dependencies:** Azure SDK for Go (`github.com/Azure/azure-sdk-for-go`)

## 5. Feature Requirements

### 5.1 Provider Kubeconfig Download (P0)

Complete the download pipeline for AWS, GCP, and Azure following the established pattern:

```
CREDENTIAL DISCOVERY ──▶ API CALL ──▶ KUBECONFIG GENERATION ──▶ WRITE TO ~/.kube/
     (done for AWS)       (needed)        (needed)                  (done)
```

Each provider must:
1. Use discovered credentials to authenticate to the provider's API
2. List all available Kubernetes clusters
3. Download or generate a kubeconfig for each cluster
4. Write to `~/.kube/{label}-kubeconfig.yaml` (existing merge pipeline handles the rest)

Follow the dispatcher pattern in `download.go` — add a `case` for each new provider.

### 5.2 Context Renaming (P1)

Replace auto-generated context names with human-readable ones:
- **Input:** Provider metadata (cluster name, region/location, resource group)
- **Output:** Context name in format `{cluster-name}@{qualifier}` where qualifier is region, location, or resource group
- **Conflict resolution:** If two clusters share a name, append a disambiguator

Implement in the existing `rename.go` stub. Run after merge, before cleanup.

### 5.3 Backup Before Merge (P1)

Before modifying `~/.kube/config`:
1. Copy to `~/.kube/config.bak.{ISO8601-timestamp}`
2. Keep the last N backups (configurable, default 5)
3. Log the backup path so users can recover

### 5.4 Dry-Run Mode (P2)

`kubectm --dry-run` should:
- Discover credentials and list available clusters
- Show what would be added/modified/removed in the kubeconfig
- Exit without modifying any files

### 5.5 Provider Extensions (P2)

Extend the Aptakube icon pattern to other providers:
- AWS EKS: EKS icon
- GCP GKE: GKE icon
- Azure AKS: AKS icon

Embed icons via `//go:embed` following the existing `lke.png` pattern.

### 5.6 Selective Sync (P3)

Allow users to exclude specific clusters:
- `~/.kubectm/excludes.json` — list of cluster identifiers to skip
- `kubectm --exclude "cluster-name"` — add to exclude list
- `kubectm --include "cluster-name"` — remove from exclude list

### 5.7 Auto-Refresh / Watch Mode (P3)

`kubectm --watch` runs on an interval (default: 30 minutes) to keep configs current. Useful as a background service or cron job.

## 6. Architecture

### 6.1 Data Flow

```
                    ┌──────────────────────────────────────────┐
                    │                kubectm                    │
                    └──────────────┬───────────────────────────┘
                                   │
                    ┌──────────────▼───────────────────────────┐
                    │         Credential Discovery              │
                    │  ┌─────────┬─────────┬────────┬────────┐ │
                    │  │ Linode  │   AWS   │  GCP   │ Azure  │ │
                    │  │  Done   │  Done   │ Needed │ Needed │ │
                    │  └────┬────┴────┬────┴───┬────┴───┬────┘ │
                    └───────┼─────────┼────────┼────────┼──────┘
                            │         │        │        │
                    ┌───────▼─────────▼────────▼────────▼──────┐
                    │           Provider Selection              │
                    │     (interactive multi-select prompt)      │
                    │   persisted to ~/.kubectm/selected_*.json │
                    └──────────────┬───────────────────────────┘
                                   │
                    ┌──────────────▼───────────────────────────┐
                    │         Kubeconfig Download                │
                    │  ┌─────────┬─────────┬────────┬────────┐ │
                    │  │ Linode  │   AWS   │  GCP   │ Azure  │ │
                    │  │  Done   │ Needed  │ Needed │ Needed │ │
                    │  └────┬────┴────┬────┴───┬────┴───┬────┘ │
                    └───────┼─────────┼────────┼────────┼──────┘
                            │         │        │        │
                            ▼         ▼        ▼        ▼
                    ┌─────────────────────────────────────────┐
                    │    ~/.kube/{label}-kubeconfig.yaml       │
                    │         (temporary files)                │
                    └──────────────┬──────────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────────┐
                    │           Backup (P1)                    │
                    │   ~/.kube/config.bak.{timestamp}         │
                    └──────────────┬──────────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────────┐
                    │           Merge Configs                   │
                    │   Merge all .yaml → ~/.kube/config       │
                    │   Handle context conflicts                │
                    │   Add provider extensions                 │
                    └──────────────┬──────────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────────┐
                    │           Rename Contexts (P1)           │
                    │   Auto-generated → human-readable        │
                    └──────────────┬──────────────────────────┘
                                   │
                    ┌──────────────▼──────────────────────────┐
                    │           Cleanup                         │
                    │   Delete temporary .yaml files            │
                    └─────────────────────────────────────────┘
```

### 6.2 Package Structure (Target)

```
kubectm/
├── cmd/main.go                     # Entry point, CLI flags, orchestration
├── pkg/
│   ├── credentials/
│   │   ├── retrieve.go             # Dispatcher: RetrieveAll, RetrieveSelected
│   │   ├── linode.go               # Done
│   │   ├── aws.go                  # Done
│   │   ├── azure.go                # Needs implementation
│   │   ├── gcp.go                  # Needs implementation
│   │   └── *_test.go
│   ├── kubeconfig/
│   │   ├── download.go             # Dispatcher: routes to provider downloaders
│   │   ├── linode.go               # Done
│   │   ├── aws.go                  # Needs implementation (EKS API)
│   │   ├── azure.go                # Needs implementation (AKS API)
│   │   ├── gcp.go                  # Needs implementation (GKE API)
│   │   ├── merge.go                # Done
│   │   ├── rename.go               # Needs implementation
│   │   ├── backup.go               # New — backup before merge
│   │   └── *_test.go
│   ├── ui/prompt.go                # Done
│   └── utils/logging.go            # Done
└── docs/
    └── PRD.md                      # This document
```

### 6.3 Adding a New Provider (Pattern)

Each provider follows a consistent pattern:

1. **`pkg/credentials/{provider}.go`** — `retrieve{Provider}Credentials() (*Credential, error)`
   - Check env vars first, then config files
   - Return `nil, nil` if not found (not an error)
   - Obfuscate credentials in logs

2. **`pkg/credentials/retrieve.go`** — Add to `RetrieveAll()` and `RetrieveSelected()` switch

3. **`pkg/kubeconfig/{provider}.go`** — `download{Provider}KubeConfig(cred Credential) error`
   - Authenticate using credential details
   - List clusters via provider API
   - Download/generate kubeconfig per cluster
   - Write to `~/.kube/{label}-kubeconfig.yaml`

4. **`pkg/kubeconfig/download.go`** — Add case to `DownloadConfigs()` switch

5. **Tests** — Unit tests for credential parsing + mock API tests for download

## 7. CLI Interface (Target)

```
kubectm [options]

Options:
  -h, --help          Show help message
  -v, --version       Show version
  --reset-creds       Reset stored credentials and prompt for new ones
  --dry-run           Show what would change without modifying files (P2)
  --exclude <name>    Exclude a cluster from sync (P3)
  --include <name>    Remove a cluster from the exclude list (P3)
  --watch [interval]  Run on interval, keeping configs current (P3)
  --backup-count <n>  Number of config backups to keep (default: 5) (P1)
```

## 8. Non-Functional Requirements

### 8.1 Security

- **No credentials in logs.** All credential values pass through `utils.ObfuscateCredential()`.
- **Path traversal protection.** All file operations validate paths are within expected directories (`~/.kube/`, `~/.kubectm/`).
- **No credential storage.** kubectm never stores cloud provider credentials — only the provider selection. Credentials are read from env vars or existing config files at runtime.
- **Minimal permissions.** Provider API calls use read-only operations only (list clusters, get kubeconfig).
- **GPG-signed releases.** All release binaries are GPG signed and attested.

### 8.2 Reliability

- **Partial failure tolerance.** If one provider fails, others still proceed. Errors are logged per-provider.
- **Idempotent merges.** Running kubectm twice produces the same `~/.kube/config`.
- **Backup before write.** Config backups ensure recoverability (P1).

### 8.3 Compatibility

- **Platforms:** Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64, arm64)
- **Go version:** 1.25+
- **Works with:** `kubectl`, `kubectx`, `kubens`, Aptakube, Lens, k9s

### 8.4 Performance

- Provider API calls are sequential per-provider (one provider at a time).
- Within a provider, cluster enumeration + download may be parallelized in future.
- Target: full sync across 4 providers with ~20 total clusters completes in under 30 seconds.

## 9. Implementation Priority

| Phase | Scope | Effort (CC) |
|-------|-------|-------------|
| **Phase 1** | AWS EKS kubeconfig download | ~30 min |
| **Phase 2** | GCP GKE credential discovery + kubeconfig download | ~45 min |
| **Phase 3** | Azure AKS credential discovery + kubeconfig download | ~45 min |
| **Phase 4** | Context renaming (all providers) | ~20 min |
| **Phase 5** | Backup before merge | ~15 min |
| **Phase 6** | Dry-run mode | ~20 min |
| **Phase 7** | Provider icon extensions | ~10 min |
| **Phase 8** | Selective sync (excludes) | ~30 min |
| **Phase 9** | Watch mode | ~30 min |

## 10. Success Metrics

- All 4 major cloud providers (Linode, AWS, GCP, Azure) fully functional
- Zero credential leakage in logs or output
- Test coverage >80% across all packages
- Single `kubectm` command syncs all providers in under 30 seconds
- Works as a drop-in complement to `kubectx`

## 11. Open Questions

1. **AWS multi-region:** Should kubectm scan all regions by default, or require a configured region list? Scanning all regions is slow (~20+ regions) but complete.
2. **GCP multi-project:** If the user has access to many GCP projects, should all be scanned? Or only the active project from `gcloud config`?
3. **Azure managed identity:** Should kubectm detect when running inside Azure and use managed identity automatically?
4. **Kubeconfig TTL:** Some providers (Linode) generate kubeconfigs with expiring tokens. Should kubectm track and auto-refresh before expiry?
5. **Plugin architecture:** Should provider support be compiled-in (current approach) or pluggable via separate binaries (like `kubectl` plugins)?
