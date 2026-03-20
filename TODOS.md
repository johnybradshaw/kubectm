# TODOS

## Provider Implementation

(No open items)

## Code Quality

(No open items)

## Completed

### P1: Implement AWS EKS kubeconfig download
**What:** Implement full AWS EKS kubeconfig download using the architecture decisions from eng review (2026-03-20).
**Why:** AWS EKS is the highest-priority missing provider. Credential discovery is done; download pipeline is the remaining gap.
**Context:** Architecture decisions locked in:
- **Region strategy (Option D):** Auto-discover enabled regions via EC2 `DescribeRegions` API. Optional override via `~/.kubectm/config.json` `aws_regions` field. If config file exists with regions, use those instead.
- **Auth:** Generate kubeconfigs with `exec` block calling `aws eks get-token` (standard EKS pattern, requires `aws` CLI installed).
- **Parallelism:** Use `sync/errgroup` with bounded concurrency (~5 goroutines) for parallel region scanning.
- **Timeout:** `context.WithTimeout(30s)` wrapping the entire AWS download flow.
- **Error handling:** Per-region errors logged and skipped (partial success). Only fail the whole provider if DescribeRegions fails AND no config override exists.
- **Context naming:** `{cluster-name}@{region}` (e.g., `prod-cluster@us-east-1`).
- **Files to create/modify:** New `pkg/kubeconfig/aws.go`, add `case "AWS"` to `download.go`.
- **Tests:** 14 test cases covering region discovery, cluster listing, kubeconfig generation, parallel orchestration, and all error paths. Use `httptest.NewServer` to mock AWS APIs.
**Completed:** v0.1.0 (2026-03-20)

### P2: Refactor credential obfuscation in retrieve.go
**What:** Extract the copy-pasted credential obfuscation + logging pattern into a shared helper like `logCredentialDiscovery(provider string, cred *Credential)`. Also fix two related issues found in eng review (2026-03-20).
**Why:** Same 5-line pattern repeated for every provider. With 4 providers, this is 4x duplication. Also uses raw `log.Printf` instead of the project's `utils.*Logger` pattern — and `utils.init()` redirects `log` output to `io.Discard`, so **these log statements are currently silently discarded** (bug).
**Completed:** v0.1.0 (2026-03-20)

### P2: Fix ObfuscateCredential for short credentials
**What:** Fix `ObfuscateCredential()` in `utils/logging.go` to return `****` for credentials <= 8 chars instead of returning them in cleartext.
**Why:** Latent security issue — line 36-37 returns short credentials unmasked.
**Completed:** v0.1.0 (2026-03-20)
