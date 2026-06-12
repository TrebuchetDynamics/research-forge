# Decision resolution checklist

Use this checklist when an owner decision unblocks one of the remaining `TODO.md` items.

## Before implementation

1. Run `make todo-audit` and copy the relevant decision ID.
2. Open an Owner decision issue using `.github/ISSUE_TEMPLATE/owner_decision.yml` or a prefilled draft in `docs/decisions/`.
3. Record the approved option, approver, date, and blocked TODO lines.

## During implementation

1. Update or add the approved artifact (`LICENSE`, Fyne dependency/screens, or superseding ADR).
2. Keep core behavior in shared services and add/adjust tests first.
3. Update `TODO.md` only for items actually implemented by the approved decision.
4. Update `docs/remaining-todo-audit.md` and `rforge decisions` if any unchecked items remain.

## Before merge

```sh
make check
go test ./...
go vet ./...
git diff --check
```

The PR must link the Owner decision issue and include evidence that `rforge decisions --check TODO.md` still passes if unchecked items remain.
