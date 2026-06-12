# Owner decisions needed

These items remain intentionally open in `TODO.md` because they require an explicit owner/product decision before implementation. See [remaining-todo-audit.md](remaining-todo-audit.md) for the prompt-to-artifact mapping of every unchecked item.

When resolving one of these blockers, open an `Owner decision` issue using `.github/ISSUE_TEMPLATE/owner_decision.yml` so the selected option, blocked TODO lines, and implementation steps are recorded before code changes. Run `make decisions` or `rforge --json decisions` to print the current machine-readable blocker list. Follow [decision-resolution-checklist.md](decision-resolution-checklist.md) when implementing an approved decision.

Issue-body scaffolds can be generated with:

```sh
make decision-issues
# or individually:
rforge decisions --issue-body project_license
rforge decisions --issue-body fyne_desktop_build_scope
```

Prefilled drafts are also stored in:

- [decisions/project_license_issue.md](decisions/project_license_issue.md)
- [decisions/fyne_desktop_build_scope_issue.md](decisions/fyne_desktop_build_scope_issue.md)

## Project license

Decision ID: `project_license`

`TODO.md`: Add license after owner decision.

Decision needed:

- choose a project license (for example MIT, Apache-2.0, GPL-family, source-available, or no public license yet);
- confirm copyright holder text;
- update `README.md` license section and add `LICENSE` if a license is selected.

See [license-decision.md](license-decision.md) for option trade-offs and implementation steps.

## Fyne desktop build scope

Decision ID: `fyne_desktop_build_scope`

`TODO.md`: Fyne dependency/screens and Fyne final MVP items.

Current decision: ADR 0005 defers the Fyne dependency until desktop build ownership is explicit. `rforge --json ui` reports this deferral for automation and release audits.

Decision needed:

- supported desktop platforms;
- packaging target and CI expectations;
- whether to add `fyne.io/fyne/v2` now or keep dependency-free view models only;
- visual QA/smoke-test scope for project create/open, search/library, and OSS dashboards.

See [fyne-desktop-plan.md](fyne-desktop-plan.md) for the implementation slices to run after the build decision is made.
