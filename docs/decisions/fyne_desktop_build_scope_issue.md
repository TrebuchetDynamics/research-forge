# Owner decision issue: fyne_desktop_build_scope

## Decision ID

fyne_desktop_build_scope

## Blocked TODO items

- Add Fyne dependency after build decision
- Add Fyne search screen
- Add Fyne library screen
- Create/open a research project from the Fyne UI
- View OSS repository studies in Fyne

## Options considered

- Defer Fyne dependency: keep current dependency-free view models and CLI-first MVP.
- Add Fyne now for all desktop MVP screens: higher dependency/build/CI scope.
- Add Fyne behind build tags: allows opt-in desktop work with lower default CI impact.

## Decision

- Pending desktop platform, packaging, CI, and visual-QA ownership decision.

## Implementation steps after approval

- Update or supersede ADR 0005.
- Add `fyne.io/fyne/v2` if approved.
- Wire widgets to `internal/ui` view models and shared services.
- Implement project create/open, search/library, and OSS study screens.
- Add Fyne smoke/build validation.
- Mark the Fyne TODO and final Fyne MVP items complete.
- Run `go test ./...`, `go vet ./...`, and `git diff --check`.
