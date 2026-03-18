# CLAUDE.md — credentials Module

## Module Purpose

Discovers and retrieves cloud provider credentials from environment variables and local config files. Each provider has its own retrieval function that returns a `*Credential` struct.

## Key Files

| File | Description |
|------|-------------|
| `retrieve.go` | Central dispatcher: `RetrieveAll()` and `RetrieveSelected()` |
| `linode.go` | Reads `LINODE_ACCESS_TOKEN` env var or `~/.config/linode-cli` |
| `aws.go` | Reads `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` env vars or `~/.aws/credentials` |
| `azure.go` | Stub — not yet implemented |
| `gcp.go` | Stub — not yet implemented |
| `aws_test.go` | Tests for AWS credential retrieval |

## Dependencies

### Internal
- `pkg/utils` — logging helpers

### External
- Standard library only (no external dependencies)

## Conventions

- Each provider gets its own file named `<provider>.go`
- Credential discovery follows a chain: env vars first, then config files
- Sensitive values are obfuscated in logs via `utils.ObfuscateCredential()`
- Return errors rather than calling `log.Fatal`; let `cmd/main.go` handle fatal errors
