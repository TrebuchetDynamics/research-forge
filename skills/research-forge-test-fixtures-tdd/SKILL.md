---
name: research-forge-test-fixtures-tdd
description: Build ResearchForge deterministic test fixtures and harnesses with TDD. Use for mock scholarly APIs, fake projects, golden files, generated PDFs, TEI fixtures, fake git repos, R/metafor fixtures, or end-to-end test data.
---

# ResearchForge Test Fixtures TDD

Use this skill to create safe deterministic fixtures before implementing behavior that depends on external data or services.

## Quick start

1. Name the behavior the fixture must prove.
2. Add the smallest legal fixture and a failing test that consumes it.
3. Implement fixture loader/helper only as needed.
4. Keep fixture provenance/license notes with the fixture.
5. Refactor to reusable builders when duplication appears.

## TDD contract

- **Red:** failing test that needs a fixture/harness.
- **Green:** minimal fixture/helper to make the test pass.
- **Refactor:** stabilize paths, sorting, timestamps, and IDs for deterministic output.
- **Receipt:** targeted tests and fixture inventory note.

## Fixture categories

- Mock OpenAlex/Crossref/arXiv/Unpaywall responses.
- Generated public test PDFs.
- Mock GROBID TEI XML.
- Parsed passage/document golden files.
- Fake project workspaces.
- Fake git repositories for OSS clone tests.
- Screening decision histories.
- Evidence extraction schemas and tables.
- R/metafor input/output fixtures.
- Report golden outputs.

## Verification gate

Done requires:

- fixture is legal to commit;
- fixture is minimal and deterministic;
- tests fail without the fixture/helper;
- timestamps/random IDs are normalized or fixed.

## Red lines

- Do not commit copyrighted PDFs or private research data.
- Do not use live APIs as fixtures.
- Do not create giant fixtures when a tiny synthetic example proves the behavior.
- Do not hide fixture source/license ambiguity.

## References

- [Fixture policy](references/fixture-policy.md)
