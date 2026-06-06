---
name: research-forge-foundation-tdd
description: Build ResearchForge Go project foundation with strict TDD. Use for Go module setup, rforge CLI skeleton, project workspace, manifest, lockfile, provenance log, SQLite storage, doctor command, or CI foundation.
---

# ResearchForge Foundation TDD

Use this skill for Milestone 0 foundation work in ResearchForge.

## Quick start

1. Read `DEVELOPMENT_PLAN.md` Milestone 0 and relevant PRD sections.
2. Pick one small behavior slice.
3. Write the failing test first.
4. Implement only enough production code to pass.
5. Refactor while tests stay green.

## TDD contract

Every slice must follow red-green-refactor:

- **Red:** add or update a failing Go test, golden fixture, or CLI integration test that proves the missing behavior.
- **Green:** implement the smallest production change that passes the test.
- **Refactor:** simplify package boundaries, names, or duplication without changing behavior.
- **Receipt:** run `go test ./...` or the narrowest equivalent plus any CLI smoke command.

Do not write production foundation code before a failing test exists unless the change is non-executable scaffolding such as directories, docs, or static assets.

## Foundation slice order

1. Go module and package layout.
2. `rforge --help` and version command.
3. JSON output convention for CLI commands.
4. Project create/open/list.
5. `rforge.project.toml` manifest read/write.
6. `rforge.lock.json` lockfile read/write.
7. Append-only provenance event log.
8. SQLite project database initialization.
9. `rforge doctor` checks.
10. CI validation commands.

## Verification gate

Before done, provide:

- failing-test evidence from the red step;
- passing-test command output;
- changed files;
- next TDD slice.

## Red lines

- Do not skip tests because the project is early.
- Do not add UI, ingestion, or analysis features in this skill except placeholders required by foundation tests.
- Do not choose PostgreSQL/OpenSearch/Qdrant as mandatory defaults without an ADR.

## References

- [Foundation checklist](references/checklist.md)
