# Owner decision record: project_license

## Decision ID

project_license

## Status

resolved

## Tracking issue

https://github.com/TrebuchetDynamics/research-forge/issues/1

Current issue title: `Owner decision: project_license (SPDX, copyright holder, approver, date required)`

## Recommended issue routing

- Labels: `decision`, `blocked`, `owner-input-needed`
- Milestone: `Owner decisions`

## Resolved TODO items

- `TODO.md:34` — Add license after owner decision

## Owner decision (recorded)

- License SPDX identifier: MIT
- Copyright holder: Trebuchet Dynamics
- Approved by: XelHaku (repository owner)
- Approval date: 2026-06-13

The owner selected MIT for permissive, broad adoption of the ResearchForge CLI
and shared Go services. The approval was recorded on tracking issue #1 and
verified with `make license-decision-approval-gate` (`approved:true`).

## Options considered

- MIT: permissive, simple, minimal patent language. **(selected)**
- Apache-2.0: permissive with explicit patent grant.
- GPL-3.0/AGPL-3.0: copyleft options for stronger sharing requirements.
- No public license yet: preserves all rights but blocks external reuse.

## Required owner response fields

- License SPDX identifier
- Copyright holder
- Approved by
- Approval date

## Implementation (completed)

- Recorded owner approval on issue #1 and confirmed `make license-decision-approval-gate` reports `approved:true`.
- Added `LICENSE` with the MIT license text and `Copyright (c) 2026 Trebuchet Dynamics`.
- Updated `README.md` license section to name the MIT License and SPDX identifier.
- Checked off the `TODO.md` license item.
- Ran `make check`.
- Ran `rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md`.

## Outcome

The license blocker no longer gates TODO completion. No owner decisions remain
open for the TODO checklist.
