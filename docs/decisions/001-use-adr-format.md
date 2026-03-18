# ADR-001: Use Architecture Decision Records

## Status

Accepted

## Context

We need a lightweight way to capture significant architectural and technical decisions so that future contributors understand the reasoning behind choices.

## Decision Drivers

- Decisions are often lost in Slack threads or meeting notes.
- New team members need context on why things are the way they are.
- We want a format that lives alongside the code.

## Options Considered

1. **ADR files in `docs/decisions/`** — Simple markdown files, numbered sequentially.
2. **Wiki pages** — Separate from the codebase, harder to keep in sync.
3. **Inline code comments** — Too scattered, no overview.

## Decision Outcome

Option 1 — ADR files in `docs/decisions/`. Each record follows this template:

- **Status**: Proposed | Accepted | Deprecated | Superseded
- **Context**: What is the issue?
- **Decision Drivers**: What forces are at play?
- **Options Considered**: What alternatives were evaluated?
- **Decision Outcome**: What was decided and why?
- **Consequences**: What are the trade-offs?

## Consequences

- Every significant decision gets a numbered markdown file.
- Records are immutable once accepted; superseded records link to their replacement.
