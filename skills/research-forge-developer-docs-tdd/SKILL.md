---
name: research-forge-developer-docs-tdd
description: Maintain ResearchForge developer and user documentation with doc-tests or validation first. Use for CLI docs, architecture docs, ADRs, contributor guides, API docs, examples, tutorials, or generated help text.
---

# ResearchForge Developer Docs TDD

Use this skill when documentation must stay synchronized with behavior.

## Quick start

1. Identify the behavior the documentation describes.
2. Add or update a test/script that would fail if docs are stale where possible.
3. Update docs only after behavior or decision is verified.
4. Prefer executable examples and generated help snippets.
5. Record validation evidence.

## TDD contract

- **Red:** failing doc test, example command, link check, generated-doc diff, or ADR expectation.
- **Green:** update docs or generator to pass.
- **Refactor:** reduce duplicate docs and link to canonical sources.
- **Receipt:** validation commands and changed docs.

## Doc areas

- CLI command reference.
- Project format docs.
- External service setup: GROBID, OpenSearch, Qdrant, R/metafor.
- Privacy and copyright docs.
- Contributor setup.
- Architecture overview and ADR index.
- TDD workflow docs.
- Fixture policy docs.
- User tutorials with open data.

## Verification gate

Done requires:

- docs match implemented behavior or clearly mark planned behavior;
- examples are executable or intentionally marked pseudo-code;
- links are valid where checked;
- ADRs are used only for durable trade-offs.

## Red lines

- Do not document unimplemented features as available.
- Do not include real API keys, private paths, or copyrighted full text in examples.
- Do not duplicate command references manually if generated help can be used.

## References

- [Documentation validation](references/documentation-validation.md)
