---
name: research-forge-web-ui-tdd
description: Build ResearchForge local web GUI through testable view models with strict TDD. Use for project dashboard, search/library screens, paper/artifact visualization, PRISMA/citation diagrams, screening UI, evidence tables, analysis views, report browser, OSS dashboard, or CLI/UI parity.
---

# ResearchForge Web GUI TDD

Use this skill for local web GUI work across ResearchForge milestones. ADR 0006 re-scopes the former native desktop plan to a Go + HTMX local research cockpit; keep `docs/web-gui-plan.md` aligned when UI scope changes.

## Workflow

1. Confirm the shared service/view model already exists; if not, build it first with the relevant domain skill.
2. Write a failing view-model, API, DOM, or screenshot/smoke test for the UI state.
3. Keep state transitions deterministic and fixture-backed.
4. Implement minimal local web server, route, component, or static artifact-viewing code.
5. Run focused tests, then broader `go test ./...` and frontend checks once a frontend workspace exists.

## Required UI states

Every screen must cover:

- loading;
- empty;
- error;
- populated;
- provenance/source-link visibility where relevant;
- keyboard-accessible interaction paths;
- chart/table alternatives for visualizations.

## Screen backlog

- Project dashboard and project open/create.
- Search builder and result review.
- Library table/detail and paper metadata.
- PDF/section/passages view.
- Citation graph and PRISMA diagram visualization.
- Screening queue and decision panel.
- Evidence table and source-link review.
- Meta-analysis setup/result viewer for forest/funnel plots and heterogeneity outputs.
- Report/artifact browser for CLI-generated Markdown/HTML/LaTeX, diagrams, plots, and tables.
- OSS repository intelligence dashboard.

## Rules

- Do not put core research workflow logic inside browser components.
- Use Go + HTMX for local research cockpit screens; add small targeted JavaScript visualization libraries only when graph/table-heavy views require them.
- Never expose private project files outside the local process without explicit user configuration.
- Large jobs must run through cancellable shared services with progress state.
