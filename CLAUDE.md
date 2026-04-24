# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
# Build the binary (module is `kubectm`, entry point under ./cmd)
go build -o kubectm ./cmd

# Vet the code
go vet ./...

# Run tests with the same flags CI uses (.github/workflows/build.yml)
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out

# Run a single package or pattern
go test -v ./pkg/kubeconfig/...
go test -v -run TestMerge ./pkg/kubeconfig/...

# After changing imports, regenerate go.sum via modules (never hand-edit go.sum — a hook blocks it)
go mod tidy

# Cross-compile — release uses `-ldflags "-X main.Version=<ver>"`
GOOS=linux  GOARCH=amd64 go build -ldflags "-X main.Version=dev" -o kubectm-linux-amd64   ./cmd
GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=dev" -o kubectm-darwin-arm64  ./cmd
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=dev" -o kubectm-windows-amd64.exe ./cmd
```

## Architecture Overview

kubectm is a CLI tool that discovers cloud-provider credentials, downloads their kubeconfigs, and merges them into `~/.kube/config`.

### Package layout

- **cmd/main.go** — flag parsing, persistence of the selected-provider list, and the overall orchestration. Fatal errors are logged here; packages below return errors.
- **pkg/credentials/** — per-provider credential discovery (`retrieve.go` dispatches to `aws.go`, `linode.go`, stubs for `azure.go`/`gcp.go`).
- **pkg/kubeconfig/** — per-provider download (`download.go` dispatches to `linode.go`, `aws.go`), plus `merge.go` and the embedded `lke.png` for the Aptakube extension.
- **pkg/ui/prompt.go** — `survey/v2` multi-select for picking providers.
- **pkg/utils/logging.go** — shared `InfoLogger`/`WarnLogger`/`ErrorLogger`/`ActionLogger`, `Iso8601Time()`, and `ObfuscateCredential()`.

Each `pkg/*` directory has its own `CLAUDE.md` with finer-grained conventions — read those before non-trivial edits to that module.

### Runtime data flow

1. `cmd/main.go` loads the saved provider list from **`~/.kubectm/selected_providers.json`** (written by `SaveSelectedCredentialProviders`). If missing, or `--reset-creds` was passed, it falls back to `credentials.RetrieveAll()` → `ui.SelectCredentials()` → save.
   - Caveat: `--reset-creds` deletes `~/.kubectm/selected_credentials.json` (the `storedCredsPath` constant), which is **not** the file the load/save functions use — leave this as-is unless the user asks for a fix.
2. `credentials.RetrieveSelected(providers)` fails fast: any missing/failed selected provider aborts the run. `RetrieveAll` is tolerant — failures are logged and skipped.
3. `kubeconfig.DownloadConfigs` routes each `Credential` to its provider downloader; configs are written to `~/.kube/<label>-kubeconfig.yaml`.
4. `kubeconfig.MergeConfigs` merges every `*.yaml` in `~/.kube/` into `~/.kube/config`, resolves context conflicts, adds the Aptakube extension for Linode contexts, then deletes the temporary `*-kubeconfig.yaml` files.

### Provider integrations

- **Linode LKE** — `GET /v4/lke/clusters` and `/v4/lke/clusters/{id}/kubeconfig`, Bearer-token auth from `LINODE_ACCESS_TOKEN` or `~/.config/linode-cli`.
- **AWS EKS** — AWS SDK v2. Auto-discovers regions via `EC2 DescribeRegions` (overridable by `aws_regions` in `~/.kubectm/config.json`), then `EKS ListClusters` + `DescribeCluster` across up to 5 parallel regions with a 30s total timeout. Generated kubeconfigs use an `exec` block calling `aws eks get-token`; contexts are named `{cluster}@{region}`. See `TODOS.md` for the architectural decisions locked in at v0.1.0.
- **Azure / GCP** — `pkg/credentials/{azure,gcp}.go` are stubs; no downloader exists.

### Conventions that cross packages

- **Error handling** — packages return errors; only `cmd/main.go` calls `Fatal*`. `utils.init()` redirects the standard `log` package to `io.Discard`, so raw `log.Printf` output is silently dropped — always use the `utils.*Logger` variables.
- **Logging format** — every log line starts with `utils.Iso8601Time()` followed by the message.
- **Credential obfuscation** — any credential value hitting logs must go through `utils.ObfuscateCredential()`; the `logCredentialDiscovery` helper in `pkg/credentials/retrieve.go` is the canonical pattern.
- **Path-traversal guard** — anything writing under `~/.kube/` (or `~/.kubectm/`) must validate the final path with `filepath.Clean` + `strings.HasPrefix` against the expected root. `pkg/kubeconfig/merge.go` is the reference.
- **Context-merge rules** — same cluster + same context name: skip; different cluster + same context name: overwrite (documented in `pkg/kubeconfig/CLAUDE.md`).

## Project-local Claude Code extensions (`.claude/`)

- **Slash commands** (`.claude/skills/*/SKILL.md`):
  - `/add-provider <Name>` — scaffolds `pkg/credentials/<name>.go`, `pkg/kubeconfig/<name>.go`, wires both dispatchers, and updates module CLAUDE.md tables. Follow it rather than scaffolding by hand.
  - `/code-review` — diff-based review against these conventions.
  - `/refactor <target>` — behaviour-preserving refactor that runs `go test -race -count=1` before and after.
  - `/release <vX.Y.Z>` — validates preconditions, tags, and pushes to trigger CI. CI (not local) produces signatures and attestations.
- **Subagents** (`.claude/agents/`): `security-reviewer` (credential/file/API review) and `test-writer` (table-driven Go tests following existing patterns in `pkg/credentials/aws_test.go`, `pkg/kubeconfig/merge_test.go`, `pkg/kubeconfig/linode_test.go`).
- **Hooks** (`.claude/settings.json`):
  - `PostToolUse` on `Edit`/`Write`: auto-runs `gofmt -w` on `*.go`, then `go vet ./...` (first 20 lines). Expect formatting to happen after each edit.
  - `PreToolUse` blocks direct edits to `go.sum`, `*.age`, and `*.key` files — use `go mod tidy` for the first, and do not attempt the others.
  - Bash auto-allowlist covers `go test|build|vet`, `gofmt`, and `go mod tidy`.

## Test-writing conventions

- Table-driven with `t.Run` subtests; name tests `Test<Func>` / `Test<Func>_<scenario>`; helpers call `t.Helper()`; parallelise with `t.Parallel()` when safe.
- Use `t.TempDir()` for filesystem work and `t.Setenv()` for environment manipulation — never touch the real `~/.kube/` or `~/.aws/` from tests.
- External APIs are mocked with `httptest.NewServer` (see `pkg/kubeconfig/linode_test.go` and `aws_test.go`).

## CLI surface

- `-h`/`--help` — help.
- `-v`/`--version` — prints the value injected via `-ldflags -X main.Version=...` (defaults to `development`).
- `--reset-creds` — removes the stored credential-selection file and re-prompts.

## CI / Release

`.github/workflows/build.yml` runs `go test -race -coverprofile ...` on every push to `main` and on PRs. On `v*` tags it additionally runs Snyk, builds the 3×2 matrix with version-injected `ldflags`, signs the binaries with GPG (`secrets.GPG_PRIVATE_KEY`/`GPG_PASSPHRASE`), publishes a GitHub Release, and produces SLSA build-provenance attestations (`actions/attest-build-provenance`). Prefer driving releases through `/release` rather than tagging manually.
