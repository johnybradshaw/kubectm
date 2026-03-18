# Architecture Overview — kubectm

## Context

kubectm is a CLI tool that downloads and merges Kubernetes configurations from cloud providers into `~/.kube/config`. It targets DevOps engineers and platform teams who manage clusters across multiple cloud providers.

## High-Level Design

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  CLI Entry   │────▶│  Credentials     │────▶│  Kubeconfig     │
│  (cmd/)      │     │  (pkg/credentials)│     │  (pkg/kubeconfig)│
└─────────────┘     └──────────────────┘     └─────────────────┘
       │                                            │
       ▼                                            ▼
┌─────────────┐                              ┌─────────────────┐
│  UI Prompts  │                              │  ~/.kube/config  │
│  (pkg/ui)    │                              │  (merged output) │
└─────────────┘                              └─────────────────┘
```

## Component Summary

| Component | Responsibility | Key Technologies |
|-----------|---------------|-----------------|
| `cmd` | CLI entry point, flag parsing, orchestration | `flag`, `encoding/json` |
| `pkg/credentials` | Discover and retrieve cloud provider credentials | Env vars, config file parsing |
| `pkg/kubeconfig` | Download, merge, and rename kubeconfigs | `k8s.io/client-go`, Linode API |
| `pkg/ui` | Interactive provider selection prompts | `survey/v2` |
| `pkg/utils` | Shared logging and utility functions | `fatih/color` |

## Data Flow

1. CLI parses flags and loads saved provider selection from `~/.kubectm/selected_credentials.json`
2. On first run (or `--reset-creds`), credentials module discovers available providers
3. UI module prompts user to select which providers to use
4. For each selected provider, kubeconfig module downloads configs via provider APIs
5. Downloaded configs are merged into `~/.kube/config`
6. Temporary per-cluster files are cleaned up

## Cross-Cutting Concerns

### Authentication & Authorisation
Each cloud provider has its own credential discovery chain (env vars → config files). Credentials are obfuscated in log output via `utils.ObfuscateCredential()`.

### Logging & Observability
Structured loggers with coloured prefixes and ISO 8601 timestamps (`InfoLogger`, `WarnLogger`, `ErrorLogger`, `ActionLogger`).

### Error Handling
Package functions return errors; `cmd/main.go` handles fatal logging. Path traversal protection on all file operations within `~/.kube/`.

## Deployment

Distributed as standalone binaries for linux/darwin/windows (amd64/arm64). Releases are automated via GitHub Actions on version tags.
