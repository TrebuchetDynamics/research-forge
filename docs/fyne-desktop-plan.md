# Fyne desktop implementation plan

Fyne desktop work is intentionally deferred by ADR 0005 until the desktop build scope is owned. This plan records the implementation slices to run once that decision changes.

## Decision inputs

Before adding `fyne.io/fyne/v2`, decide:

- supported platforms and packaging targets;
- whether desktop CI runs on every PR or only release branches;
- visual QA expectations and screenshot/golden strategy;
- accessibility expectations for keyboard navigation and screen readers;
- whether desktop state persists only through existing project services.

## TDD slices after approval

1. Add the Fyne dependency and a minimal `rforge ui` app entry point.
2. Wire project create/open to `internal/project` services.
3. Wire dashboard, search, library, and OSS screens to `internal/ui` view models.
4. Add loading/error/empty states for each screen using existing background job abstractions.
5. Add smoke tests or build tags for Fyne package checks.
6. Keep all core logic outside widgets; widgets should adapt view models only.

## Current ready seams

- `internal/ui` has dependency-free view models for dashboard, search, library, OSS, citations, screening, evidence, analysis, and reports.
- `rforge --json ui` reports the deferral for automation.
- `make fyne-smoke` documents the current deferred smoke-check behavior.
