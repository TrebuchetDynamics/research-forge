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
| [`rforge`](./skills/rforge/SKILL.md) | **External user skill** — install into any project to give an agent rforge awareness; auto-installs rforge if missing, then handles literature search, provenance, and review packaging for any academic topic |
| [`research-forge`](./skills/research-forge/SKILL.md) | **Internal agent usage skill** — run `rforge` to conduct research and save outputs to a project folder or arbitrary path; covers literature search, OSS study, meta-analysis, and knowledge capture |
| [`research-forge-workflow-orchestration-tdd`](./skills/research-forge-workflow-orchestration-tdd/SKILL.md) | Milestone breakdown, slice sequencing, handoffs, acceptance criteria, routing to specialist skills |
| [`research-forge-architecture-tdd`](./skills/research-forge-architecture-tdd/SKILL.md) | Package boundaries, ADRs, shared core, service interfaces, adapter seams, dependency direction |
| [`research-forge-foundation-tdd`](./skills/research-forge-foundation-tdd/SKILL.md) | Go module, `rforge` CLI, project workspace, manifest, lockfile, provenance, SQLite, doctor, CI |
| [`research-forge-test-fixtures-tdd`](./skills/research-forge-test-fixtures-tdd/SKILL.md) | Mock APIs, fake projects, golden files, generated PDFs, TEI, fake git repos, analysis/report fixtures |
| [`research-forge-scholarly-ingestion-tdd`](./skills/research-forge-scholarly-ingestion-tdd/SKILL.md) | OpenAlex, Crossref, arXiv, Unpaywall, `PaperRecord`, dedupe, import/export, library workflows |
| [`research-forge-oss-intelligence-tdd`](./skills/research-forge-oss-intelligence-tdd/SKILL.md) | OSS repository catalog, clone workspace, license checks, study notes, OSS reports |
| [`research-forge-document-pipeline-tdd`](./skills/research-forge-document-pipeline-tdd/SKILL.md) | Legal PDF acquisition, GROBID parsing, passage extraction, indexing, retrieval |
| [`research-forge-screening-tdd`](./skills/research-forge-screening-tdd/SKILL.md) | Include/exclude screening, reason tags, reviewer workflow, PRISMA counts, active-learning scaffold |
| [`research-forge-evidence-extraction-tdd`](./skills/research-forge-evidence-extraction-tdd/SKILL.md) | Extraction schemas, evidence tables, source links, validation states, LLM suggestions |
| [`research-forge-meta-analysis-tdd`](./skills/research-forge-meta-analysis-tdd/SKILL.md) | Effect sizes, R/metafor adapter, analysis inputs, plots, heterogeneity, reproducibility |
| [`research-forge-reporting-tdd`](./skills/research-forge-reporting-tdd/SKILL.md) | Markdown/HTML/LaTeX reports, PRISMA diagrams, evidence/citation tables, audit appendix |
| [`research-forge-web-ui-tdd`](./skills/research-forge-web-ui-tdd/SKILL.md) | Local web GUI screens, artifact visualizations, view models, background jobs, CLI/UI parity |
| [`research-forge-data-governance-tdd`](./skills/research-forge-data-governance-tdd/SKILL.md) | Schemas, migrations, archives, privacy defaults, copyright/OA policy, provenance retention, compatibility |
| [`research-forge-quality-security-tdd`](./skills/research-forge-quality-security-tdd/SKILL.md) | Threat modeling, secrets, path safety, external commands, fuzzing, dependency scans, CI hardening |
| [`research-forge-performance-tdd`](./skills/research-forge-performance-tdd/SKILL.md) | Benchmarks, large-library performance, indexing throughput, caching, UI responsiveness, memory use |
| [`research-forge-release-packaging-tdd`](./skills/research-forge-release-packaging-tdd/SKILL.md) | Versioning, cross-platform builds, web GUI packaging, checksums, install smoke tests, release notes |
| [`research-forge-developer-docs-tdd`](./skills/research-forge-developer-docs-tdd/SKILL.md) | CLI docs, architecture docs, ADR index, contributor guides, examples, tutorials, generated help |

## Recommended usage order

1. Workflow orchestration TDD to pick the smallest vertical slice.
2. Architecture TDD only when a seam or ADR-sensitive decision is needed.
3. Test fixtures TDD before external-data/service-heavy behavior.
4. Foundation TDD for Milestone 0.
5. Scholarly ingestion TDD and OSS intelligence TDD for Milestones 1-2.
6. Document pipeline TDD, screening TDD, and evidence extraction TDD for Milestones 3-5.
7. Meta-analysis TDD and reporting TDD for Milestones 6-7.
8. Web GUI TDD alongside each milestone for parity.
9. Data governance, quality/security, performance, docs, and release skills whenever the slice crosses those concerns.

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
