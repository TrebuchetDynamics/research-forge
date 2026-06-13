---
name: research-forge-evidence-extraction-tdd
description: Build ResearchForge structured evidence extraction with strict TDD. Use for extraction schemas, manual extraction, LLM-assisted suggestions, evidence tables, source passage links, validation status, or evidence export.
---

# ResearchForge Evidence Extraction TDD

Use this skill for Milestone 5 evidence extraction.

## Quick start

1. Read `DEVELOPMENT_PLAN.md` Milestone 5 and PRD section 7.6.
2. Choose one evidence behavior with a source-passage fixture.
3. Write a failing test that requires evidence provenance.
4. Implement only enough extraction/schema code to pass.
5. Refactor while preserving audit links.

## TDD contract

- **Red:** failing test for schema validation, source link requirement, evidence status transition, export, or suggestion review behavior.
- **Green:** minimal implementation.
- **Refactor:** separate accepted evidence from machine suggestions.
- **Receipt:** run targeted tests plus relevant CLI smoke command.

## Slice order

1. Extraction schema format and validator.
2. EvidenceItem model with source support requirement.
3. Manual `rforge extract add`.
4. Status transitions: suggested, accepted, rejected, corrected.
5. Evidence table CSV/JSON/Markdown export.
6. Audit check for unsupported evidence.
7. LLM suggestion adapter interface behind explicit config.
8. Suggestion review/acceptance workflow.
9. web GUI evidence table view model.

## Verification gate

Done requires:

- accepted evidence cannot exist without source support;
- LLM suggestions are never silently accepted;
- export tests preserve IDs and provenance;
- weak or missing support is reported.

## Red lines

- Do not let generated text become a scientific claim without human acceptance.
- Do not store evidence values detached from paper/document/passage identity.
- Do not put provider secrets in fixtures, logs, or docs.

## References

- [Evidence schema notes](references/evidence-schema.md)
