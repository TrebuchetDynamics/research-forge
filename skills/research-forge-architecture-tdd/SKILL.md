---
name: research-forge-architecture-tdd
description: Shape ResearchForge architecture decisions with tests first. Use for package boundaries, ADRs, service interfaces, plugin seams, dependency direction, storage/search adapter choices, or CLI/UI shared-core design.
---

# ResearchForge Architecture TDD

Use this skill when a development slice needs an architecture decision or seam before implementation.

## Quick start

1. Read `DEVELOPMENT_PLAN.md`, `RESEARCH-FORGE-PRD.md`, and existing `docs/adr/` if present.
2. Name the behavior or constraint the architecture must protect.
3. Write a characterization or contract test first.
4. Implement the smallest seam/interface/package boundary that passes.
5. Write an ADR only for hard-to-reverse, surprising, real trade-offs.

## TDD contract

- **Red:** failing contract test, package-level test, fixture, or compile-time assertion that exposes the missing seam.
- **Green:** minimal interface/package/adapter decision to pass.
- **Refactor:** remove dependency cycles and keep business rules in shared services.
- **Receipt:** run `go test ./...` or a targeted package test plus any architecture check.

## Decision areas

- CLI framework and command topology.
- Shared application core used by CLI and web GUI.
- SQLite/PostgreSQL storage boundary.
- Local search vs OpenSearch adapter boundary.
- Optional Qdrant/vector adapter boundary.
- External-service adapters for GROBID, R/metafor, APIs, and LLMs.
- Provenance event format and replay boundaries.
- Project format compatibility and migrations.

## Verification gate

Done requires:

- tests prove the seam or constraint;
- no avoidable import cycles;
- production code does not depend on concrete external services where an adapter is planned;
- ADR created or explicitly rejected with reason.

## Red lines

- Do not write speculative abstractions without a failing test or immediate consumer.
- Do not add an ADR for obvious reversible choices.
- Do not let UI packages own core domain behavior.

## References

- [Architecture checklist](references/architecture-checklist.md)
