# TODO completion audit

This document records the current completion audit for the active `TODO.md` objective.

## Success criteria

`TODO.md` is complete only when every checked item has implementation or documentation evidence, and every unchecked item is either resolved or explicitly tracked by an owner decision or implementation issue with an executable audit.

## Prompt-to-artifact checklist

| Requirement / deliverable | Evidence | Status |
| --- | --- | --- |
| Complete repository/planning foundation items | `TODO.md` checked items, project docs, issue/PR templates, ADR index | Complete except decision-gated license |
| Add license after owner decision | GitHub issue #1, `docs/license-decision.md`, `docs/owner-decisions.md`, `README.md`, `CONTRIBUTING.md`, `.github/ISSUE_TEMPLATE/owner_decision.yml`, `.github/PULL_REQUEST_TEMPLATE.md`, `rforge decisions` JSON `blocker_kind`, `owner_action_required`, `owner_inputs`, `implementation_steps`, `todo_refs`, `issue_title`, `issue_labels`, and `milestone`, completion audit verifies `license_file_absent_when_blocked`, `license_decision_pending_verified`, `license_decision_draft_owner_inputs_verified`, `license_decision_required_response_fields_verified`, `license_owner_approval_absent_verified`, `readme_license_pending_verified`, `license_owner_inputs_verified`, `license_owner_response_fields_verified`, `license_options_verified`, `license_implementation_steps_verified`, `license_issue_routing_verified`, `license_issue_title_verified`, `remaining_todo_audit_verified`, `license_decision_brief_verified`, `owner_decisions_license_section_verified`, `owner_decision_template_verified`, `owner_decision_template_response_fields_verified`, `pr_license_gate_verified`, and `contributing_license_workflow_verified` while pending | Blocked by owner decision |
| Add Go + HTMX web GUI workspace/dependencies | `internal/webui`, `web/assets/researchforge.css`, ADR 0006, `docs/web-gui-plan.md`, `rforge --json ui` | Complete |
| Add web GUI search screen | `internal/webui.NewSearchHandler`, `internal/webui/search_test.go`, dependency-free search view model, ADR 0006 | Complete |
| Add web GUI library screen | `internal/webui.NewLibraryHandler`, `internal/webui/library_test.go`, dependency-free library view model, ADR 0006 | Complete |
| Create/open a research project from the web GUI | `internal/webui.NewProjectHandler`, `NewCreateProjectHandler`, `NewOpenProjectHandler`, `internal/webui/project_test.go`, CLI/shared project services | Complete |
| View OSS repository studies in the web GUI | `internal/webui.NewOSSHandler`, `internal/webui/oss_test.go`, OSS services and view model, ADR 0006 | Complete |
| View CLI-generated papers, meta-analysis outputs, PRISMA/citation diagrams, and report artifacts in the web GUI | `internal/webui.NewArtifactsHandler`, `internal/webui/artifacts_test.go`, CLI-generated report/analysis/diagram artifacts, ADR 0006, `docs/web-gui-plan.md` (Go + HTMX/"Go HTMLX"), `SKILLS.md`, `skills/research-forge-web-ui-tdd/SKILL.md` | Complete |
| Add Go + HTMX web GUI smoke-check target | `Makefile` target `web-gui-smoke` runs `go test ./internal/webui` | Complete |
| Verify unchecked TODO coverage | `rforge decisions --check TODO.md` verifies TODO coverage, line refs, and issue refs | Passing when `make todo-audit` passes |
| Verify closeout checklist coverage | `rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md` verifies decision/tracker coverage plus this prompt-to-artifact checklist; JSON includes `completion_blocked`, `blocked_decisions`, `blocked_decision_ids`, `license_decision_required_response_fields_verified`, `license_owner_approval_absent_verified`, `license_owner_response_fields_verified`, `owner_decision_template_response_fields_verified`, `license_options_verified`, `license_issue_routing_verified`, `license_issue_title_verified`, and `remaining_todo_audit_verified` so passing audit output is not mistaken for completed decision-gated work | Passing when `make todo-completion-audit` passes |
| Verify full local quality gate | `make check` runs `go test ./...`, `go vet ./...`, TODO completion audit, and `git diff --check` | Required before marking implementation complete |

## Latest validation evidence

As of 2026-06-12, local closeout checks were run against the current tree:

- `make check` passed: `go test ./...`, `go vet ./...`, TODO completion audit, and `git diff --check`.
- `make todo-audit` passed and reported one unchecked line, `TODO.md:34`, covered by the `project_license` owner decision.
- Root license-file scan found no `LICENSE*` or `COPYING*` file, which is expected while the owner license decision is pending.
- GitHub issue #1 was inspected with `gh issue view 1`: it is open, titled `Owner decision: project_license (SPDX, copyright holder, approver, date required)`, assigned to `XelHaku`, routed with exactly the `decision`, `blocked`, and `owner-input-needed` labels plus the `Owner decisions` milestone, and contains no owner approval response yet.
- GitHub issue #1 was updated with the current closeout blocker summary and required owner response fields: https://github.com/TrebuchetDynamics/research-forge/issues/1#issuecomment-4693487934.
- GitHub issue #1 body was regenerated from `rforge decisions --issue-body project_license`; a live `gh issue view 1` check verified it contains the current issue title, `TODO.md:34`, the required owner response fields, `make license-decision-approval-gate`, and `approved:true` gating instructions.
- `make license-decision-live-audit` reported `approved:false` for issue #1; `make license-decision-approval-gate` is therefore expected to fail until owner approval is recorded.
- `make license-decision-approval-gate` was run against the current issue state and failed with `approval_gate_exit=2`, confirming the license TODO remains blocked.
- GitHub issue #1 was updated with the latest approval-gate unblock instructions: https://github.com/TrebuchetDynamics/research-forge/issues/1#issuecomment-4694416877.
- A strict `gh issue view 1` audit confirmed the issue contains the required field names but no non-placeholder license response (`License SPDX identifier: MIT|Apache-2.0|GPL-3.0-*|AGPL-3.0-*|NOASSERTION`), no non-placeholder copyright holder, no non-placeholder approver, and no non-placeholder `Approval date: YYYY-MM-DD`; the issue remains open and blocked.

## Current conclusion

The planning direction now selects Go + HTMX as the only primary local web GUI stack, and the tracked web GUI implementation slices from issue #2 are complete. Do not mark the full implementation checklist complete until issue #1 is resolved and the remaining license `TODO.md` item is implemented and checked off or explicitly re-scoped by an approved decision.
