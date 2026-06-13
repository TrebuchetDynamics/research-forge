# Owner decisions needed

These items remain intentionally open in `TODO.md` because they require an explicit owner/product decision before implementation. See [remaining-todo-audit.md](remaining-todo-audit.md) for the prompt-to-artifact mapping of every unchecked item.

When resolving one of these blockers, open or reuse an `Owner decision` issue using `.github/ISSUE_TEMPLATE/owner_decision.yml` so the selected option, blocked TODO lines, and implementation steps are recorded before code changes. Run `make decisions` or `rforge --json decisions` to print the current machine-readable blocker list, including `issue_labels`, `milestone`, `options_considered`, and `owner_response_required_fields`, and `make todo-completion-audit` before closeout. Run `make license-decision-live-audit` to inspect issue #1 for non-placeholder owner approval fields, and `make license-decision-approval-gate` when you expect approval to be complete; it must pass with `approved:true` before adding `LICENSE`. Run `make todo-completion-audit` or `rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md` before claiming TODO closeout; the audit reports whether completion remains blocked and verifies `license_owner_response_fields_verified`, `license_options_verified`, plus `license_issue_routing_verified` for the license blocker. Follow [decision-resolution-checklist.md](decision-resolution-checklist.md) when implementing an approved decision.

Issue-body scaffolds can be generated with:

```sh
make decision-issues
# or individually:
rforge decisions --issue-body project_license
```

Prefilled drafts are also stored in:

- [decisions/project_license_issue.md](decisions/project_license_issue.md)

Resolved implementation tracker records are stored in:

- [decisions/web_gui_stack_scope_issue.md](decisions/web_gui_stack_scope_issue.md)

Open tracking issues:

- [#1 Owner decision: project_license (SPDX, copyright holder, approver, date required)](https://github.com/TrebuchetDynamics/research-forge/issues/1)

## Project license

Decision ID: `project_license`

Status: `owner_decision_required`

Tracking issue: [#1 Owner decision: project_license (SPDX, copyright holder, approver, date required)](https://github.com/TrebuchetDynamics/research-forge/issues/1)

`TODO.md`: Add license after owner decision.

Decision needed:

- choose a project license (for example MIT, Apache-2.0, GPL-family, source-available, or no public license yet);
- confirm copyright holder text;
- record the approving owner and approval date;
- update `README.md` license section and add `LICENSE` if a license is selected.

See [license-decision.md](license-decision.md) for option trade-offs and implementation steps.

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
