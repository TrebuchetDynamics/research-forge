# Contributing to ResearchForge

ResearchForge is developed test-first. Contributions should be small, auditable, and aligned with the project principle:

> Retrieval-first, provenance-first, statistics-first, LLM-assisted.

## Before you start

Read:

- [README.md](./README.md)
- [RESEARCH-FORGE-PRD.md](./RESEARCH-FORGE-PRD.md)
- [DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md)
- [ROADMAP.md](./ROADMAP.md)
- [TODO.md](./TODO.md)
- [SKILLS.md](./SKILLS.md)

## TDD requirement

Every production-code change must follow red-green-refactor:

1. **Red** — add or update a failing test, golden fixture, benchmark, or integration test.
2. **Green** — implement only enough code to pass.
3. **Refactor** — simplify while tests stay green.
4. **Receipt** — include validation commands and results in the pull request.

Documentation-only changes should still include validation where practical, such as link checks, generated help diffs, or `git diff --check`.

## Slice size

Prefer one observable behavior per pull request. Good slices look like:

- `rforge project create` writes a manifest and provenance event.
- OpenAlex fixture response normalizes one work into `PaperRecord`.
- `screen decide` rejects an unknown exclusion reason.
- Markdown report output includes a search-query audit section.

Avoid broad slices like "build ingestion" or "add UI".

## Safety rules

Do not commit:

- secrets, tokens, cookies, credentials, or `.env` files;
- private research data;
- copyrighted PDFs or full text without explicit license clearance;
- local clone workspaces under `opensource/clones/`;
- generated build artifacts or local machine state.

Use mocked APIs and deterministic fixtures for normal tests. Live network or external-service tests must be opt-in.

## Provenance and reproducibility

If a change affects user-visible research workflow, it should preserve or add provenance. Ask:

- What was searched or imported?
- Where did each record/document come from?
- What exact source supports each extracted claim?
- What tool versions, parameters, and outputs reproduce the result?

## Pull requests

A pull request should include:

- summary of behavior;
- TDD receipt;
- validation commands;
- privacy/copyright impact;
- provenance/reproducibility impact;
- linked TODO or roadmap item.
