---
name: research-forge-reporting-tdd
description: Build ResearchForge reproducible report generation with strict TDD. Use for Markdown, HTML, LaTeX, PRISMA diagrams, bibliography tables, evidence tables, audit appendices, or report export workflows.
---

# ResearchForge Reporting TDD

Use this skill for Milestone 7 report generation.

## Quick start

1. Read `DEVELOPMENT_PLAN.md` Milestone 7 and PRD sections 7.8 and 13.
2. Select one report artifact behavior.
3. Write a failing golden test or fixture-based report test.
4. Implement the smallest renderer/exporter change.
5. Refactor templates while preserving deterministic output.

## TDD contract

- **Red:** failing golden test for Markdown/HTML/LaTeX, PRISMA output, evidence table, citation table, or audit appendix.
- **Green:** minimal renderer code.
- **Refactor:** keep rendering separate from data collection and provenance queries.
- **Receipt:** run golden tests and relevant CLI smoke command.

## Slice order

1. Report data model assembled from a fixture project.
2. Markdown report skeleton.
3. Citation/bibliography table.
4. Evidence table.
5. Screening/PRISMA summary.
6. Analysis artifact section.
7. Audit appendix.
8. HTML export.
9. LaTeX export scaffold.
10. web GUI report builder view model.

## Verification gate

Done requires:

- deterministic report output under golden tests;
- report answers the PRD's audit questions;
- citations and evidence links are not dropped;
- generated artifacts include build metadata.

## Red lines

- Do not include unsupported claims in generated reports.
- Do not hide failed searches, parser warnings, excluded papers, or analysis warnings from the audit appendix.
- Do not make report output depend on current wall-clock time except in explicit build metadata.

## References

- [Report audit checklist](references/report-audit-checklist.md)
