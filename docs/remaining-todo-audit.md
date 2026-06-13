# Remaining TODO audit

This audit maps every unchecked `TODO.md` item to its current blocker or implementation tracker and existing evidence.

| TODO item | Blocker / tracker | Evidence |
| --- | --- | --- |
| Add license after owner decision (`TODO.md:34`) | Requires owner license choice, copyright holder text, approver, and approval date | GitHub issue [#1](https://github.com/TrebuchetDynamics/research-forge/issues/1), titled `Owner decision: project_license (SPDX, copyright holder, approver, date required)`, plus latest blocker reminder [#issuecomment-4693487934](https://github.com/TrebuchetDynamics/research-forge/issues/1#issuecomment-4693487934), `docs/owner-decisions.md`, `docs/license-decision.md`, `docs/decisions/project_license_issue.md`, `README.md` license section, `CONTRIBUTING.md`, `.github/ISSUE_TEMPLATE/owner_decision.yml`, `.github/PULL_REQUEST_TEMPLATE.md`, `rforge decisions` JSON `blocker_kind`, `owner_action_required`, `owner_inputs`, `owner_response_required_fields`, `implementation_steps`, `issue_labels`, and `milestone`; completion audit verifies `license_file_absent_when_blocked`, `license_decision_pending_verified`, `license_decision_draft_owner_inputs_verified`, `license_decision_required_response_fields_verified`, `license_owner_approval_absent_verified`, `readme_license_pending_verified`, `license_owner_inputs_verified`, `license_owner_response_fields_verified`, `license_options_verified`, `license_implementation_steps_verified`, `license_issue_routing_verified`, `license_issue_title_verified`, `remaining_todo_audit_verified`, `license_decision_brief_verified`, `owner_decisions_license_section_verified`, `owner_decision_template_verified`, `owner_decision_template_response_fields_verified`, `pr_license_gate_verified`, and `contributing_license_workflow_verified` while pending |

## Current validation evidence

Run `make todo-audit` to verify every unchecked TODO is decision/tracker-covered, verify decision line references, verify tracking issue references, print the machine-readable decision/tracker list, and show the current unchecked TODO lines. Run `make license-decision-live-audit` to inspect the live issue #1 owner approval fields; the aggregate `approved` field must be `true` before implementing the license TODO. Run `make license-decision-approval-gate` as the fail-fast gate before adding `LICENSE`. Run `make todo-completion-audit` to also verify the `docs/todo-completion-audit.md` Prompt-to-artifact checklist used for goal closeout. The JSON closeout verifier reports `completion_blocked`, `blocked_decisions`, `blocked_decision_ids`, `license_decision_required_response_fields_verified`, `license_owner_approval_absent_verified`, `license_owner_response_fields_verified`, `owner_decision_template_response_fields_verified`, `license_options_verified`, `license_issue_routing_verified`, `license_issue_title_verified`, and `remaining_todo_audit_verified` so passing audit output is not mistaken for completed decision-gated work. Use `make decisions-markdown` or `rforge decisions --markdown` to print a markdown decision/evidence table. When a decision or implementation slice is approved, follow [decision-resolution-checklist.md](decision-resolution-checklist.md) before marking any remaining TODO complete.

The current local validation gate is:

```sh
make check
```

`make check` currently runs:

```sh
go test ./...
go vet ./...
go run ./cmd/rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md
git diff --check
```

The remaining TODO should not be checked off until the owner/license decision is complete and the corresponding license file/docs are added or explicitly re-scoped.
