# ResearchForge Development Skills

This repository includes project-specific Pi skills for developing ResearchForge. Every implementation skill requires TDD: write a failing test first, make it pass with the smallest production change, then refactor while tests remain green.

## Universal TDD rule

For all ResearchForge development:

1. **Red** — create a failing test, golden fixture, or integration test that proves the missing behavior.
2. **Green** — implement only enough production code to pass.
3. **Refactor** — improve structure without changing behavior.
4. **Receipt** — record the validation command and result.

Production code changes that skip the red step are not accepted unless they are non-executable scaffolding, documentation, generated assets, or emergency build fixes explicitly called out as such.

## Skill inventory

| Skill | Use for |
|---|---|
| [`research-forge-foundation-tdd`](./skills/research-forge-foundation-tdd/SKILL.md) | Go module, `rforge` CLI, project workspace, manifest, lockfile, provenance, SQLite, doctor, CI |
| [`research-forge-scholarly-ingestion-tdd`](./skills/research-forge-scholarly-ingestion-tdd/SKILL.md) | OpenAlex, Crossref, arXiv, Unpaywall, `PaperRecord`, dedupe, import/export, library workflows |
| [`research-forge-oss-intelligence-tdd`](./skills/research-forge-oss-intelligence-tdd/SKILL.md) | OSS repository catalog, clone workspace, license checks, study notes, OSS reports |
| [`research-forge-document-pipeline-tdd`](./skills/research-forge-document-pipeline-tdd/SKILL.md) | Legal PDF acquisition, GROBID parsing, passage extraction, indexing, retrieval |
| [`research-forge-screening-tdd`](./skills/research-forge-screening-tdd/SKILL.md) | Include/exclude screening, reason tags, reviewer workflow, PRISMA counts, active-learning scaffold |
| [`research-forge-evidence-extraction-tdd`](./skills/research-forge-evidence-extraction-tdd/SKILL.md) | Extraction schemas, evidence tables, source links, validation states, LLM suggestions |
| [`research-forge-meta-analysis-tdd`](./skills/research-forge-meta-analysis-tdd/SKILL.md) | Effect sizes, R/metafor adapter, analysis inputs, plots, heterogeneity, reproducibility |
| [`research-forge-reporting-tdd`](./skills/research-forge-reporting-tdd/SKILL.md) | Markdown/HTML/LaTeX reports, PRISMA diagrams, evidence/citation tables, audit appendix |
| [`research-forge-fyne-ui-tdd`](./skills/research-forge-fyne-ui-tdd/SKILL.md) | Fyne desktop UI screens, view models, background jobs, CLI/UI parity |

## Recommended usage order

1. Foundation TDD
2. Scholarly ingestion TDD
3. OSS intelligence TDD
4. Document pipeline TDD
5. Screening TDD
6. Evidence extraction TDD
7. Meta-analysis TDD
8. Reporting TDD
9. Fyne UI TDD alongside each milestone for parity

## Development handoff format

When handing off a slice, include:

```text
TDD slice: <behavior>
Red: <test added and failing evidence>
Green: <implementation summary>
Refactor: <cleanup summary or none>
Validation: <commands/results>
Next slice: <recommended next failing test>
```
