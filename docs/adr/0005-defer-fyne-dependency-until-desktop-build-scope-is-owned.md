# ADR 0005: Defer Fyne dependency until desktop build scope is owned

## Status

Superseded by [ADR 0006](0006-rescope-fyne-desktop-to-local-web-gui.md) for future UI direction.

## Context

ResearchForge has implemented shared application services and UI-ready view models for dashboard, search, library, OSS, citation, screening, evidence, analysis, and report flows. The remaining desktop work is Fyne-specific: adding the dependency, packaging, widget screens, and project open/create flows.

Adding Fyne now would expand the dependency graph and CI/build requirements before desktop ownership, packaging targets, visual QA, and platform support are explicitly scoped.

## Decision

ResearchForge deferred adding the Fyne dependency until the desktop build scope was owned. ADR 0006 later re-scoped the primary UI to a local web GUI, and the Go + HTMX implementation slices have landed. Under the superseding decision:

- core behavior stays in shared services;
- `internal/ui` provides dependency-free view models;
- `internal/webui` adapts those models into local Go + HTMX screens;
- the historical Fyne package smoke check remains documented as deferred.

## Consequences

- CLI and service behavior can continue to harden without desktop dependency churn.
- superseding Go + HTMX local web GUI TODO items are complete and covered by `make web-gui-smoke`.
- Future primary UI work should follow ADR 0006 and wire web views to existing view models instead of duplicating core logic.
