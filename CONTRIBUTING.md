# Contributing to ResearchForge

ResearchForge is developed test-first. Contributions should be small, auditable, and aligned with the project principle:

> Retrieval-first, provenance-first, statistics-first, LLM-assisted.

## Before you start

Read:

- [README.md](./README.md)
- [RESEARCH-FORGE-PRD.md](./RESEARCH-FORGE-PRD.md)
- [DEVELOPMENT_PLAN.md](./DEVELOPMENT_PLAN.md)
- [ROADMAP.md](./ROADMAP.md)
- [CHECKLIST.md](./CHECKLIST.md)
- [SKILLS.md](./SKILLS.md)

## Local prerequisites and checks

Before running the core gate, have Go 1.26 (the `go.mod` toolchain is Go 1.26.6), Git, GNU Make, and Node.js >=22 available. Node is required for the adapter tests, not just the optional documentation site.

Run `make check` for the local gate: Go formatting/module checks, Go and Node adapter tests, vet, TODO completion audit, inventory check, and whitespace checks. See [Developer setup](./docs/developer-setup.md) for cold-cache provisioning and the separate optional site requirements (Node >=22.12 and npm >=9.6.5).

## TDD requirement

Every production-code change must follow red-green-refactor:

1. **Red** — add or update a failing test, golden fixture, benchmark, or integration test.
2. **Green** — implement only enough code to pass.
3. **Refactor** — simplify while tests stay green.
4. **Receipt** — include validation commands and results in the pull request.

Documentation-only changes should still include validation where practical, such as link checks, generated help diffs, or `git diff --check`.

## Slice size

Prefer one observable behavior per pull request. Good slices look like:

- `rforge project create` writes a manifest and provenance event.
- OpenAlex fixture response normalizes one work into `PaperRecord`.
- `screen decide` rejects an unknown exclusion reason.
- Markdown report output includes a search-query audit section.

Avoid broad slices like "build ingestion" or "add UI".

## Safety rules

Do not commit:

- secrets, tokens, cookies, credentials, or `.env` files;
- private research data;
- copyrighted PDFs or full text without explicit license clearance;
- local clone workspaces under `opensource/clones/`;
- generated build artifacts or local machine state.

Use mocked APIs and deterministic fixtures for normal tests. Live network or external-service tests must be opt-in.

## Provenance and reproducibility

If a change affects user-visible research workflow, it should preserve or add provenance. Ask:

- What was searched or imported?
- Where did each record/document come from?
- What exact source supports each extracted claim?
- What tool versions, parameters, and outputs reproduce the result?

## Research-quality improvement claims

Improvement claims must cite the [engine-tier baseline](./docs/benchmark/baseline-2026-10-19.md)
and its [canonical JSON](./docs/benchmark/baseline-2026-10-19.json), with signed
measured deltas (new minus baseline), metric/K and macro/micro aggregation. Include
both engine versions/revisions, set checksums, snapshot digests and protocol
hashes. Compare implementations on identical locked inputs; changed inputs need
a newly approved set and baseline, not an implementation-win claim.

Tune on `dev`, not held-out outcomes. If detailed held-out results guide a change,
mark that set version exposed and obtain a newly approved held-out set for fresh
claims. Unavailable metrics remain null; captured replay timing is not evidence
of faster code. See the baseline's reproduction commands and the
[benchmark specification](./docs/benchmark/spec.md) for scoring and limitations.

## Decision-gated TODOs

No active owner decision currently blocks the `CHECKLIST.md` TODO checklist. Project licensing was resolved as MIT on 2026-06-13; see [Owner decisions](./docs/owner-decisions.md) for the recorded approval and completed GUI decision. This does not establish 1.0 release readiness.

For future decision-gated scope changes:

1. Run `make todo-audit` and inspect `rforge --json decisions` for active decision IDs.
2. Open or reuse an Owner decision issue with `.github/ISSUE_TEMPLATE/owner_decision.yml`. Use `rforge decisions --issue-body <active-decision-id>` only for an ID in the active registry; resolved IDs are rejected. The existing `make decision-issues` target still requests resolved `project_license` and is not a current onboarding command.
3. Link the approved decision in the PR before implementation. For license changes, confirm the approved SPDX identifier, exact copyright holder, approver, and approval date before changing `LICENSE`.
4. Preserve the live issue #1 revalidation policy: `make license-decision-live-audit` and `make license-decision-approval-gate` are required for license/release approval, not routine offline setup. See [Developer setup](./docs/developer-setup.md#license-approval-revalidation) and the [1.0 release gate](./docs/1.0-release.md).
5. Keep `rforge decisions --check CHECKLIST.md` passing if unchecked TODOs remain.
6. Run `make todo-completion-audit` or `rforge decisions --completion-audit CHECKLIST.md docs/todo-completion-audit.md` when changing TODO closeout evidence.

## Pull requests

A pull request should include:

- summary of behavior;
- TDD receipt;
- validation commands;
- privacy/copyright impact;
- provenance/reproducibility impact;
- linked TODO or roadmap item.
