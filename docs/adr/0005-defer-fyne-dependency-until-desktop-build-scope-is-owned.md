# ADR 0005: Defer Fyne dependency until desktop build scope is owned

## Status

Accepted

## Context

ResearchForge has implemented shared application services and UI-ready view models for dashboard, search, library, OSS, citation, screening, evidence, analysis, and report flows. The remaining desktop work is Fyne-specific: adding the dependency, packaging, widget screens, and project open/create flows.

Adding Fyne now would expand the dependency graph and CI/build requirements before desktop ownership, packaging targets, visual QA, and platform support are explicitly scoped.

## Decision

ResearchForge will defer adding the Fyne dependency until the desktop build scope is owned. Until then:

- core behavior stays in shared services;
- `internal/ui` provides dependency-free view models;
- `rforge ui` is a placeholder entry point;
- Fyne package smoke checks are documented as deferred.

## Consequences

- CLI and service behavior can continue to harden without desktop dependency churn.
- Fyne TODO items remain open until the build decision changes.
- Future Fyne work should wire widgets to existing view models instead of duplicating core logic.
