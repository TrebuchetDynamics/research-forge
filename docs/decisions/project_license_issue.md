# Owner decision issue: project_license

## Decision ID

project_license

## Blocked TODO items

- Add license after owner decision

## Options considered

- MIT: permissive, simple, minimal patent language.
- Apache-2.0: permissive with explicit patent grant.
- GPL-3.0/AGPL-3.0: copyleft options for stronger sharing requirements.
- No public license yet: preserves all rights but blocks external reuse.

## Decision

- Pending owner selection.

## Implementation steps after approval

- Add `LICENSE` with selected license text and copyright holder.
- Update `README.md` license section.
- Update contribution guidance if contributor terms change.
- Mark the license item in `TODO.md` complete.
- Run `go test ./...`, `go vet ./...`, and `git diff --check`.
