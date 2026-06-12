# Remaining TODO audit

This audit maps every unchecked `TODO.md` item to its current blocker and existing evidence.

| TODO item | Blocker | Evidence |
| --- | --- | --- |
| Add license after owner decision | Requires owner license choice and copyright holder text | `docs/owner-decisions.md`, `docs/license-decision.md`, `docs/decisions/project_license_issue.md`, `README.md` license section, `rforge decisions` |
| Add Fyne dependency after build decision | Requires desktop build/platform/CI ownership decision | ADR 0005, `docs/owner-decisions.md`, `docs/fyne-desktop-plan.md`, `docs/decisions/fyne_desktop_build_scope_issue.md`, `rforge --json ui`, `rforge decisions` |
| Add Fyne search screen | Blocked by Fyne dependency/build decision | `internal/ui` dependency-free search view model, ADR 0005, `docs/fyne-desktop-plan.md` |
| Add Fyne library screen | Blocked by Fyne dependency/build decision | `internal/ui` dependency-free library view model, ADR 0005, `docs/fyne-desktop-plan.md` |
| Create/open a research project from the Fyne UI | Blocked by Fyne dependency/build decision | CLI/shared project services implemented; `rforge --json ui` reports deferral; ADR 0005 |
| View OSS repository studies in Fyne | Blocked by Fyne dependency/build decision | OSS CLI/shared services and `internal/ui` OSS dashboard view model implemented; ADR 0005 |

## Current validation evidence

Run `make todo-audit` to verify every unchecked TODO is decision-covered, verify decision line references, print the machine-readable decision list, and show the current unchecked TODO lines. Use `make decisions-markdown` or `rforge decisions --markdown` to print a markdown decision/evidence table. When a decision is approved, follow [decision-resolution-checklist.md](decision-resolution-checklist.md) before marking any remaining TODO complete.

The current local validation gates are:

```sh
go test ./...
go vet ./...
git diff --check
```

The remaining TODOs should not be checked off until the owner/build decisions above are explicitly made and the corresponding files or Fyne implementation are added.
