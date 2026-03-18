---
name: code-review
description: "Review Go code changes for correctness, security, and adherence to kubectm project conventions."
disable-model-invocation: true
---

# code-review

Review recent code changes against kubectm project conventions and Go best practices.

## Steps

1. **Gather changes**: Run `git diff` to see unstaged changes, or `git diff HEAD~1` if everything is committed.

2. **Review against conventions** (from CLAUDE.md):
   - Error handling: package functions return errors, no `log.Fatal` outside `cmd/main.go`
   - Logging: uses predefined loggers (`InfoLogger`, `WarnLogger`, `ErrorLogger`, `ActionLogger`) with ISO 8601 timestamps
   - Credentials: sensitive values obfuscated via `utils.ObfuscateCredential()`
   - File operations: paths validated within `~/.kube/`
   - New providers: follow the per-file pattern (`<provider>.go` in both `pkg/credentials/` and `pkg/kubeconfig/`)

3. **Check for common issues**:
   - Missing error checks on file/network operations
   - Credential values appearing in log output
   - Path traversal vulnerabilities
   - Missing test coverage for new functions

4. **Run verification**:
   ```bash
   go vet ./...
   go test -race ./...
   ```

5. **Report findings** grouped by severity (Critical, High, Medium, Low).

## Examples

```
/code-review
```
