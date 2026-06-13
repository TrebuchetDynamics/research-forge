# ADR 0006: Rescope Fyne desktop delivery to a local web GUI

## Status

Accepted

## Context

ResearchForge originally planned a native Fyne desktop UI, and ADR 0005 deferred that dependency until desktop build ownership existed. The product direction now favors a local browser-based GUI because the primary interactive need is visualization and review of project artifacts produced by the CLI: paper collections, citation graphs, PRISMA diagrams, evidence tables, meta-analysis plots/tables, and report outputs.

A web GUI better fits graph-heavy and report-heavy workflows, allows reuse of mature visualization libraries, and keeps the CLI as the reproducible source of truth. Its final purpose is a local research cockpit: a local human review and navigation layer over ResearchForge projects and CLI-generated artifacts. The frontend stack is Go + HTMX: a compact Go-served local app with progressive enhancement and targeted visualization libraries for graph-heavy views.

## Decision

ResearchForge will replace the planned Fyne desktop UI with a Go + HTMX local research cockpit launched by `rforge ui`.

- The Go CLI and shared Go services remain the authoritative workflow engine.
- The web GUI reads project state and CLI-generated artifacts from the local project workspace for human review, navigation, and small local workflow actions.
- Go + HTMX is the selected stack for a compact local Go server UI with progressive enhancement; SvelteKit and Astro are rejected alternatives for the primary UI path.
- Core logic must stay in Go services and dependency-free view models, not browser components.
- The implementation plan lives in `docs/web-gui-plan.md`.

## Consequences

- ADR 0005 is superseded for future UI implementation direction.
- No Fyne, SvelteKit, or Astro dependency should be added for the primary UI path unless a later ADR supersedes this decision.
- Existing UI-ready view models remain useful as web GUI adapters.
- The UI implementation tracker remains `web_gui_stack_scope`, now recording the accepted Go + HTMX stack instead of the former Fyne scope.
- `rforge ui` should evolve into a local web server/static app launcher.
- Cockpit scope explicitly includes project open/create, papers, citation graphs, PRISMA diagrams, meta-analysis artifacts, reproducible report outputs, and guided local actions backed by shared Go services.
