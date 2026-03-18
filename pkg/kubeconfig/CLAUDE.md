# CLAUDE.md — kubeconfig Module

## Module Purpose

Downloads kubeconfigs from cloud provider APIs and merges them into `~/.kube/config`. Handles context naming conflicts and cleanup of temporary files.

## Key Files

| File | Description |
|------|-------------|
| `download.go` | Dispatcher routing to provider-specific downloaders |
| `linode.go` | Calls Linode API v4 to fetch LKE cluster kubeconfigs |
| `merge.go` | Merges `.yaml` files from `~/.kube/` into main config |
| `rename.go` | Stub for renaming clusters/contexts in kubeconfigs |
| `lke.png` | Embedded Linode icon for Aptakube extension (`//go:embed`) |

## Dependencies

### Internal
- `pkg/credentials` — provider credential structs
- `pkg/utils` — logging helpers

### External
- `k8s.io/client-go` — kubeconfig handling
- `k8s.io/apimachinery` — Kubernetes API types

## Conventions

- Path traversal protection: all file operations validate paths are within `~/.kube/`
- Context conflict resolution: same-cluster contexts are skipped; different-cluster same-name contexts are overwritten
- Temporary files are downloaded to `~/.kube/{label}-kubeconfig.yaml` and deleted after merge
