# TODOS

## Provider Implementation

### P1: Decide AWS multi-region scanning strategy
**What:** Decide whether kubectm scans all AWS regions by default or requires a configured region list.
**Why:** Scanning all ~20 regions is slow (~30s) but complete. Scanning configured regions is fast (~2s) but requires user setup. This blocks AWS EKS implementation.
**Context:** The PRD lists this as an open question (Section 11, Q1). Options: (A) scan all regions with parallel goroutines, (B) use `AWS_DEFAULT_REGION` + optional `~/.kubectm/config.json` region list, (C) auto-discover enabled regions via EC2 DescribeRegions API. Option C is the best balance — fast, complete, no user config.
**Effort:** S (human) → S (CC)
**Depends on:** Nothing — must be resolved before AWS EKS download implementation.

## Code Quality

### P2: Refactor credential obfuscation in retrieve.go
**What:** Extract the copy-pasted credential obfuscation + logging pattern into a shared helper like `logCredentialDiscovery(provider string, cred *Credential)`.
**Why:** Same 5-line pattern repeated for every provider. With 4 providers, this is 4x duplication. Also uses raw `log.Printf` instead of the project's `utils.*Logger` pattern.
**Context:** See `pkg/credentials/retrieve.go` lines 25-30 (AWS), 38-43 (Azure), 50-55 (GCP), 63-67 (Linode). All follow the same shape: build obfuscated map, log it.
**Effort:** S (human) → S (CC)
**Depends on:** Nothing.

## Completed
