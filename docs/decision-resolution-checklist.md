# Decision resolution checklist

Use this checklist when an owner decision or implementation tracker unblocks one of the remaining `TODO.md` items.

## Before implementation

1. Run `make todo-audit` and copy the relevant decision ID.
2. Open or reuse an existing open issue for the Owner decision issue or implementation tracker using `.github/ISSUE_TEMPLATE/owner_decision.yml` or a prefilled draft in `docs/decisions/`.
3. Record the approved option or implementation slice, approver where applicable, date, and blocked TODO lines.
4. For `project_license`, confirm `rforge --json decisions` still exposes the owner-decision `issue_labels`, `milestone`, `options_considered`, and `owner_response_required_fields` metadata.
5. For `project_license`, record the SPDX identifier and exact copyright holder before adding `LICENSE`.
6. For `project_license`, inspect the live owner issue before implementation and confirm it contains non-placeholder owner approval fields:

   ```sh
   make license-decision-live-audit
   make license-decision-approval-gate
   # or inspect directly:
   gh issue view 1 --json body,comments,state
   ```

   Do not treat placeholder template text as approval; require non-placeholder values for license SPDX identifier, copyright holder, approver, and approval date. `make license-decision-approval-gate` must pass before adding `LICENSE`.

## During implementation

1. Update or add the approved artifact (`LICENSE`, Go + HTMX web GUI templates/screens, or superseding ADR).
2. Keep core behavior in shared services and add/adjust tests first.
3. Update `TODO.md` only for items actually implemented by the approved decision.
4. Update `docs/remaining-todo-audit.md` and `rforge decisions` if any unchecked items remain.

## Before merge

```sh
make check
go test ./...
go vet ./...
rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md
git diff --check
```

The PR must link the Owner decision or implementation tracker issue and include evidence that `rforge decisions --check TODO.md` and `rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md` still pass if unchecked items remain. Inspect `completion_blocked`, `blocked_decisions`, `blocked_decision_ids`, `license_owner_response_fields_verified`, `license_options_verified`, and `license_issue_routing_verified` before claiming the TODO objective is complete. For `project_license`, include the `gh issue view 1` live issue evidence showing non-placeholder owner approval fields before adding `LICENSE`.
