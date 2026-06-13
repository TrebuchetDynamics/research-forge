# License decision brief

## Decision recorded (2026-06-13)

ResearchForge is licensed under the **MIT License** (SPDX: `MIT`), Copyright (c)
2026 Trebuchet Dynamics, approved by the repository owner (XelHaku) on issue #1.
`make license-decision-approval-gate` reports `approved:true`, `LICENSE` carries
the MIT text, and the `README.md` license section names the license. The brief
below is retained as the rationale and process record for that decision.

## Background

Before the decision, ResearchForge had no project license and the repository was treated as all-rights-reserved by default.

## Common options

Use the SPDX identifier when recording the decision so downstream packaging, SBOM, and repository scanners can identify the license unambiguously.

| Option | SPDX identifier | Typical fit | Trade-offs |
| --- | --- | --- | --- |
| MIT | `MIT` | permissive library/CLI adoption | minimal patent language |
| Apache-2.0 | `Apache-2.0` | permissive adoption with patent grant | longer license text |
| GPL-3.0 | `GPL-3.0-only` or `GPL-3.0-or-later` | strong copyleft research tooling | limits proprietary redistribution |
| AGPL-3.0 | `AGPL-3.0-only` or `AGPL-3.0-or-later` | network-copyleft service deployments | strongest adoption constraints |
| No public license yet | `NOASSERTION` / all rights reserved note | owner wants more review | external contributors/users have no reuse grant |

## Owner inputs needed

- intended adoption model: academic, commercial, internal, or mixed;
- patent posture and contributor expectations;
- copyright holder string;
- whether dependencies and example assets are compatible with the chosen license;
- whether dual licensing or a contributor license agreement is desired.

## Required owner response fields

Before adding `LICENSE` or checking off `TODO.md:34`, record all of these fields in issue #1:

- License SPDX identifier;
- Copyright holder;
- Approved by;
- Approval date.

## Implementation after decision

1. Run `make license-decision-live-audit` and confirm issue #1 has non-placeholder owner approval fields.
2. Run `make license-decision-approval-gate`; it must pass with `approved:true`.
3. Add `LICENSE` with the selected license text.
4. Update `README.md` license section.
5. Update contribution guidance if contributor terms change.
6. Mark `TODO.md` license item complete.
