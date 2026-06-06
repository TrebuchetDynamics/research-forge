---
name: research-forge-fyne-ui-tdd
description: Build ResearchForge Fyne desktop UI through testable view models with strict TDD. Use for project dashboard, search/library screens, PDF/section view, screening UI, evidence tables, analysis views, report builder, OSS dashboard, or CLI/UI parity.
---

# ResearchForge Fyne UI TDD

Use this skill for Fyne UI work across ResearchForge milestones.

## Quick start

1. Read the relevant milestone in `DEVELOPMENT_PLAN.md` and PRD section 7.10.
2. Identify the shared service behavior already covered by CLI/domain tests.
3. Write a failing view-model or presenter test before adding widgets.
4. Implement minimal Fyne binding/widget code.
5. Refactor to keep long-running work off the UI thread.

## TDD contract

- **Red:** failing test for view model state, command dispatch, validation, background-job status, or UI parity behavior.
- **Green:** minimal implementation; prefer testing view models over fragile visual assertions.
- **Refactor:** move business logic out of widgets and into shared services.
- **Receipt:** run relevant tests and a manual UI smoke note when widgets change.

## UI slice order

1. App shell and project dashboard.
2. Project open/create flow.
3. Search form and results table.
4. Library table/detail pane.
5. PDF/section/passages view.
6. Screening queue and decision panel.
7. Evidence extraction table.
8. Analysis setup/results view.
9. Report builder/export view.
10. OSS repository intelligence dashboard.

## Verification gate

Done requires:

- tested view model behavior;
- domain/service behavior remains shared with CLI;
- background jobs expose progress/error states;
- UI changes have a manual smoke command or note.

## Red lines

- Do not put core research workflow logic inside Fyne widgets.
- Do not block the UI thread with network, parsing, indexing, or analysis jobs.
- Do not implement UI-only behavior that cannot be reproduced or audited from shared services.

## References

- [UI testing pattern](references/ui-testing-pattern.md)
