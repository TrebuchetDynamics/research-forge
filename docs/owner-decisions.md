# Owner decisions

## Current status

No active owner decision currently blocks the `CHECKLIST.md` TODO checklist. The CLI's active registry is empty; `make decisions` or `rforge --json decisions` reports `decisions: []`. This is not a declaration of 1.0 release readiness or resolution of unrelated project blockers.

Resolved decision and implementation tracker records are stored in:

- [decisions/project_license_issue.md](decisions/project_license_issue.md)
- [decisions/web_gui_stack_scope_issue.md](decisions/web_gui_stack_scope_issue.md)

See [remaining-todo-audit.md](remaining-todo-audit.md) for the prompt-to-artifact mapping and [todo-completion-audit.md](todo-completion-audit.md) for recorded closeout evidence.

## Future decision-gated work

For a new blocker, open or reuse an `Owner decision` issue using [the owner-decision template](../.github/ISSUE_TEMPLATE/owner_decision.yml) so the selected option, blocked TODO lines, and implementation steps are recorded before code changes. The `owner_decision_required` status, `issue_labels`, `milestone`, `options_considered`, and `owner_response_required_fields` describe active decision records, not the resolved records below. Follow [decision-resolution-checklist.md](decision-resolution-checklist.md) when implementing an approved decision.

Issue-body scaffolding accepts only an active ID returned by `rforge --json decisions`:

```text
rforge decisions --issue-body <active-decision-id>
```

There are no active IDs today. Resolved `project_license` and `web_gui_stack_scope` IDs are rejected. The existing `make decision-issues` recipe still requests `project_license`, so it is not a working onboarding/scaffolding command for the current registry.

Run `make todo-completion-audit` or `rforge decisions --completion-audit CHECKLIST.md docs/todo-completion-audit.md` before claiming TODO closeout. Historical license-blocker validation included `license_owner_response_fields_verified`, `license_options_verified`, and `license_issue_routing_verified`; the current resolved-license audit instead verifies `license_resolution_verified` and reports `completion_blocked` false with empty `blocked_decision_ids`.

## Project license

Decision ID: `project_license`

Status: `resolved`

Tracking issue: [#1 Owner decision: project_license (SPDX, copyright holder, approver, date required)](https://github.com/TrebuchetDynamics/research-forge/issues/1)

`CHECKLIST.md`: Add license after owner decision (resolved).

Decision recorded (2026-06-13):

- License SPDX identifier: MIT;
- Copyright holder: Trebuchet Dynamics;
- Approved by: XelHaku (repository owner);
- Approval date: 2026-06-13.

The historical record states that the owner approved MIT on issue #1 and that `make license-decision-approval-gate` passed with `approved:true` before adding `LICENSE` and closing `CHECKLIST.md:34`. `LICENSE` and the `README.md` license section name the MIT License and SPDX identifier. See [license-decision.md](license-decision.md) for the option trade-offs that informed the decision.

For license changes and release revalidation, retain `make license-decision-live-audit` to inspect issue #1 for non-placeholder owner approval fields and `make license-decision-approval-gate` to require `approved:true`. These require authorized live GitHub access; this documentation update did not rerun them. See the [1.0 release gate](1.0-release.md) for release criteria and approval fields.

## Local web GUI stack and scope

Decision ID: `web_gui_stack_scope`

Status: `complete`

Tracking issue: [#2](https://github.com/TrebuchetDynamics/research-forge/issues/2)

Current decision: ADR 0006 replaces the planned Fyne desktop UI with a Go + HTMX local research cockpit and selects **Go + HTMX** as the only primary UI stack. SvelteKit and Astro were considered but rejected for the primary UI path because they add frontend build/scope overhead or are too static for the required project workflows. `rforge --json ui` reports the selected stack and ready state for automation and release audits.

Completed implementation:

- local Go + HTMX shell/static workspace;
- search and library screens;
- project create/open forms backed by shared project services;
- OSS repository studies dashboard;
- papers, PRISMA/citation diagrams, meta-analysis outputs, report artifact views, and guided local review actions;
- `make web-gui-smoke` coverage for `internal/webui` handlers.

See [web-gui-plan.md](web-gui-plan.md) for the implementation slices.
