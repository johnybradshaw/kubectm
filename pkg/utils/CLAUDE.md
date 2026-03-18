# CLAUDE.md — utils Module

## Module Purpose

Shared utility functions used across the project, primarily logging and credential obfuscation.

## Key Files

| File | Description |
|------|-------------|
| `logging.go` | Predefined loggers (`InfoLogger`, `WarnLogger`, `ErrorLogger`, `ActionLogger`) with coloured prefixes and ISO 8601 timestamps; `ObfuscateCredential()` for masking sensitive values |

## Dependencies

### External
- `github.com/fatih/color` — coloured terminal output
