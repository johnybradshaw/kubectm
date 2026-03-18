---
name: add-provider
description: "Scaffold a new cloud provider integration for kubectm with credential retrieval and kubeconfig download."
disable-model-invocation: true
---

# add-provider

Scaffold the files and wiring needed to add a new cloud provider to kubectm.

## Inputs

Parse arguments: `/add-provider <ProviderName>` (e.g., `/add-provider DigitalOcean`)

| Parameter | Description | Required | Default |
|-----------|-------------|----------|---------|
| `ProviderName` | Display name of the provider (e.g., `DigitalOcean`) | Yes | None |

## Steps

1. **Derive names** from the provider name:
   - File name: lowercase, no spaces (e.g., `digitalocean`)
   - Provider constant: as provided (e.g., `DigitalOcean`)
   - Function suffix: PascalCase (e.g., `DigitalOcean`)

2. **Create credential retrieval** at `pkg/credentials/<provider>.go`:
   - Function signature: `func retrieve<Provider>Credentials() (*Credential, error)`
   - Follow the pattern in `pkg/credentials/aws.go`:
     - Check environment variables first
     - Fall back to config file if applicable
     - Return `&Credential{Provider: "<ProviderName>", Details: map[string]string{...}}`
     - Use `kubectm/pkg/utils` for any logging
   - Ask the user which env vars and config file paths to check

3. **Create kubeconfig downloader** at `pkg/kubeconfig/<provider>.go`:
   - Function signature: `func download<Provider>KubeConfig(cred credentials.Credential) error`
   - Follow the pattern in `pkg/kubeconfig/linode.go`:
     - Use the credential details to authenticate API calls
     - List clusters, then download kubeconfig for each
     - Write files to `~/.kube/<label>-kubeconfig.yaml`
     - Validate all file paths are within `~/.kube/` (path traversal protection)
   - Ask the user for the provider's API endpoints

4. **Wire into dispatchers** — update these existing files:

   **`pkg/credentials/retrieve.go`**:
   - Add discovery block in `RetrieveAll()` following the existing pattern (retrieve → obfuscate → append)
   - Add case in `RetrieveSelected()` switch statement

   **`pkg/kubeconfig/download.go`**:
   - Add case in `DownloadConfigs()` switch statement

5. **Create test file** at `pkg/credentials/<provider>_test.go`:
   - Table-driven tests for credential retrieval
   - Test env var discovery and config file parsing
   - Use `t.Setenv()` for environment variable tests
   - Use `t.TempDir()` for config file tests

6. **Update documentation**:
   - Add provider to `pkg/credentials/CLAUDE.md` key files table
   - Add provider to `pkg/kubeconfig/CLAUDE.md` key files table

7. **Verify**:
   ```bash
   go vet ./...
   go test -race ./...
   ```

## Example

```
/add-provider DigitalOcean
/add-provider Vultr
/add-provider Hetzner
```
