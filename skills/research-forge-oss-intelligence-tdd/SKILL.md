---
name: research-forge-oss-intelligence-tdd
description: Build ResearchForge OSS repository intelligence with strict TDD. Use for rforge oss add/clone/scan/note/license-check/report, open-source clone workspace, repository inventory, license metadata, and integration notes.
---

# ResearchForge OSS Intelligence TDD

Use this skill for Milestone 2 OSS repository study workflows.

## Quick start

1. Read `DEVELOPMENT_PLAN.md` Milestone 2 and PRD sections 4.9, 4.10, and 7.11.
2. Pick one OSS workflow behavior.
3. Write a failing test using a local fake repository or fixture metadata.
4. Implement minimal registry/CLI behavior.
5. Refactor to keep clones out of source control and notes deterministic.

## TDD contract

- **Red:** failing test for registry mutation, clone path safety, license detection, note creation, or report output.
- **Green:** smallest implementation using local test repos or mocked API responses.
- **Refactor:** separate external clone operations from committed inventory/notes.
- **Receipt:** run targeted tests and a CLI smoke command where available.

## Slice order

1. `opensource/README.md` and `.gitignore` rules for `opensource/clones/`.
2. OSS repository registry domain model.
3. `rforge oss add <owner/repo>`.
4. Safe clone path resolution and shallow clone command planning.
5. `rforge oss clone` using local/fake remotes in tests.
6. License detection from repository files.
7. Study-note template.
8. `rforge oss report` markdown output.
9. web GUI OSS dashboard view-model hook.

## Verification gate

Done requires:

- tests prove clones are not committed inventory;
- path traversal and malformed repo names are rejected;
- license/report output is deterministic;
- provenance records the study action.

## Red lines

- Do not copy external source code into ResearchForge implementation.
- Do not commit local clones.
- Do not treat license detection as legal advice.
- Do not run network clone tests by default.

## References

- [OSS safety rules](references/oss-safety.md)
