# Owner decision issue: project_license

## Decision ID

project_license

## Status

owner_decision_required

## Tracking issue

https://github.com/TrebuchetDynamics/research-forge/issues/1

Current issue title: `Owner decision: project_license (SPDX, copyright holder, approver, date required)`

## Recommended issue routing

- Labels: `decision`, `blocked`, `owner-input-needed`
- Milestone: `Owner decisions`

## Blocked TODO items

- `TODO.md:34` — Add license after owner decision

## Options considered

- MIT: permissive, simple, minimal patent language.
- Apache-2.0: permissive with explicit patent grant.
- GPL-3.0/AGPL-3.0: copyleft options for stronger sharing requirements.
- No public license yet: preserves all rights but blocks external reuse.

## Owner inputs needed

- License choice with SPDX identifier: `MIT`, `Apache-2.0`, `GPL-3.0-only`/`GPL-3.0-or-later`, `AGPL-3.0-only`/`AGPL-3.0-or-later`, `NOASSERTION`/all-rights-reserved note, or another named license.
- Intended adoption model: academic, commercial, internal, or mixed.
- Patent posture and contributor expectations.
- Exact copyright holder string.
- Whether dual licensing or a contributor license agreement is desired.

## Required owner response fields

- License SPDX identifier
- Copyright holder
- Approved by
- Approval date

## Owner response template

```md
## Decision
- License SPDX identifier: <MIT | Apache-2.0 | GPL-3.0-only | GPL-3.0-or-later | AGPL-3.0-only | AGPL-3.0-or-later | NOASSERTION | other>
- Copyright holder: <exact legal/name string>
- Adoption model: <academic | commercial | internal | mixed>
- Patent/contributor posture: <notes>
- Dual licensing / CLA expectations: <none or details>
- Approved by: <owner>
- Approval date: <YYYY-MM-DD>
```

## Decision

- Pending owner selection.

## Implementation steps after approval

- Run `make license-decision-approval-gate` and require `approved:true`.
- Add `LICENSE` with the approved SPDX license text and copyright holder.
- Update `README.md` license section.
- Update contribution guidance if contributor terms change.
- Update `TODO.md` license checkbox.
- Run `make check`.
- Run `rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md` if unchecked TODOs remain.
