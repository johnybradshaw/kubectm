---
name: refactor
description: "Refactor Go code for clarity and maintainability while preserving behaviour. Runs tests before and after to verify correctness."
disable-model-invocation: true
---

# refactor

Refactor specified Go code while preserving all existing behaviour.

## Inputs

Parse arguments: `/refactor <target>` where target is a file path, package name, or function name.

| Parameter | Description | Required | Default |
|-----------|-------------|----------|---------|
| `target` | File, package, or function to refactor | Yes | None |

## Steps

1. **Read the target code** and understand its current behaviour.

2. **Run tests before refactoring** to establish a baseline:
   ```bash
   go test -race -count=1 ./...
   ```

3. **Identify improvements** — focus on:
   - Reducing duplication
   - Improving naming clarity
   - Simplifying control flow
   - Extracting reusable helpers (only if used more than once)
   - Improving error messages

4. **Apply changes** incrementally, one logical change at a time.

5. **Run tests after each change** to verify behaviour is preserved:
   ```bash
   go test -race -count=1 ./...
   ```

6. **Summarise** what was changed and why.

## Examples

```
/refactor pkg/credentials/aws.go
/refactor pkg/kubeconfig/merge.go
```
