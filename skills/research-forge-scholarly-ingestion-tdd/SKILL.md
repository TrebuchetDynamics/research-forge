---
name: research-forge-scholarly-ingestion-tdd
description: Build ResearchForge scholarly search, source connectors, library storage, deduplication, and import/export with strict TDD. Use for OpenAlex, Crossref, arXiv, Unpaywall, PaperRecord, BibTeX, RIS, CSV, JSON, or library workflows.
---

# ResearchForge Scholarly Ingestion TDD

Use this skill for Milestone 1 metadata, source connector, and research-library work.

## Quick start

1. Read `DEVELOPMENT_PLAN.md` Milestone 1 and the source-specific PRD sections.
2. Define one observable ingestion behavior.
3. Write a failing test with mocked HTTP or fixture input.
4. Implement the smallest connector/library behavior.
5. Refactor normalization and provenance boundaries after tests pass.

## TDD contract

Required loop for every production change:

- **Red:** failing test for request construction, response normalization, persistence, dedupe, import, export, or CLI output.
- **Green:** minimal code to pass using deterministic fixtures or mocked servers; no live API dependency in tests.
- **Refactor:** isolate source-specific payloads from normalized `PaperRecord` and preserve provenance.
- **Receipt:** run targeted tests plus `go test ./...` when practical.

## Slice order

1. Source connector interface.
2. OpenAlex request and response normalization.
3. PaperRecord storage.
4. Library list/search CLI output.
5. arXiv connector.
6. Crossref connector.
7. Unpaywall OA lookup.
8. DOI/arXiv/title-author-year dedupe.
9. BibTeX/CSV/JSON import/export.
10. web GUI library/search view-model hooks.

## Verification gate

Done requires:

- a failing test was observed first;
- no test requires network access;
- source query and imported record provenance are recorded;
- CLI JSON output remains stable or golden-tested.

## Red lines

- Do not hit live scholarly APIs in normal tests.
- Do not store only source-specific raw metadata without normalized fields.
- Do not drop source payload/provenance during dedupe.
- Do not fetch copyrighted full text in this skill; use document-pipeline skill.

## References

- [Connector testing notes](references/connector-testing.md)
