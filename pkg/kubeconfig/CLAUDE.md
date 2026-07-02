# CLAUDE.md — kubeconfig Module

## Module Purpose

Downloads kubeconfigs from cloud provider APIs and merges them into `~/.kube/config`. Handles context naming conflicts and cleanup of temporary files.

## Key Files

| File | Description |
|------|-------------|
| `download.go` | Dispatcher routing to provider-specific downloaders (Linode, AWS) |
| `linode.go` | Calls Linode API v4 to fetch LKE cluster kubeconfigs |
| `aws.go` | Downloads EKS kubeconfigs via AWS SDK v2 (EC2 DescribeRegions, EKS ListClusters/DescribeCluster) with parallel region scanning, exec-based auth, and optional config override |
| `gcp.go` | Downloads GKE kubeconfigs via the GKE REST API; self-contained OAuth token flows (service-account JWT / authorized-user refresh); exec auth via `gke-gcloud-auth-plugin` |
| `azure.go` | Downloads AKS kubeconfigs via ARM (client-credentials token, managedClusters list with pagination, listClusterUserCredential) |
| `dryrun.go` | `DryRunConfigs()`: lists clusters per provider and reports what a merge would change without writing files |
| `merge.go` | Merges `.yaml` files from `~/.kube/` into main config |
| `backup.go` | Backs up `~/.kube/config` to `config.bak.{timestamp}` before merge; prunes old backups keeping the most recent N (default 5) |
| `rename.go` | Stub for renaming clusters/contexts in kubeconfigs |
| `lke.png` | Embedded Linode icon for Aptakube extension (`//go:embed`) |
| `linode_test.go` | Tests for Linode kubeconfig download |
| `aws_test.go` | Tests for AWS EKS kubeconfig download (13 cases with httptest mock servers) |
| `gcp_test.go` | Tests for GCP token flows and GKE kubeconfig download (httptest mock servers) |
| `azure_test.go` | Tests for Azure token flow and AKS kubeconfig download (httptest mock servers) |
| `dryrun_test.go` | Tests for dry-run listing and no-write guarantees |
| `backup_test.go` | Tests for kubeconfig backup and pruning |

## Dependencies

### Internal
- `pkg/credentials` — provider credential structs
- `pkg/utils` — logging helpers

### External
- `k8s.io/client-go` — kubeconfig handling
- `k8s.io/apimachinery` — Kubernetes API types
- `github.com/aws/aws-sdk-go-v2` — AWS EC2/EKS API calls

## Conventions

- Path traversal protection: all file operations validate paths are within `~/.kube/`
- Context conflict resolution: same-cluster contexts are skipped; different-cluster same-name contexts are overwritten
- Temporary files are downloaded to `~/.kube/{label}-kubeconfig.yaml` and deleted after merge
