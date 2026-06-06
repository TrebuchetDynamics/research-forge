---
name: research-forge-document-pipeline-tdd
description: Build ResearchForge legal full-text acquisition, PDF assets, GROBID parsing, passage extraction, and indexing with strict TDD. Use for Unpaywall PDF fetch, document assets, parsed sections, references, OpenSearch, Qdrant, Bleve, or retrieval.
---

# ResearchForge Document Pipeline TDD

Use this skill for Milestone 3 document acquisition, parsing, and indexing.

## Quick start

1. Read `DEVELOPMENT_PLAN.md` Milestone 3 and PRD sections 4.3, 7.12, and 8.3.
2. Choose one document-pipeline behavior with a small legal fixture.
3. Write a failing test before adding parser, asset, or index code.
4. Implement with mocked services or tiny fixtures.
5. Refactor around provenance and copyright-safe asset handling.

## TDD contract

- **Red:** failing test for OA policy, asset checksum, parser adapter, passage IDs, index update, or retrieval provenance.
- **Green:** minimal behavior using mock HTTP, mock GROBID TEI, or local fixtures.
- **Refactor:** isolate external service clients from core document model.
- **Receipt:** run targeted tests; integration tests requiring services must be opt-in.

## Slice order

1. DocumentAsset model with checksum and legal acquisition metadata.
2. OA URL selection from Unpaywall metadata.
3. PDF fetch command with mocked HTTP.
4. GROBID adapter interface.
5. TEI fixture parsing into sections/references/passages.
6. Passage ID stability tests.
7. Local full-text index MVP.
8. Optional OpenSearch adapter.
9. Optional Qdrant adapter.
10. Retrieval CLI with exact source passage references.

## Verification gate

Done requires:

- tests do not rely on copyrighted PDFs;
- every passage/retrieval result has source provenance;
- parser/service version can be recorded in the lockfile or event log;
- failed parsing is observable and recoverable.

## Red lines

- Do not download or commit copyrighted PDFs.
- Do not make GROBID/OpenSearch/Qdrant mandatory for local MVP without an accepted ADR.
- Do not return retrieval snippets without paper/section/passage identity.

## References

- [Document fixtures](references/document-fixtures.md)
