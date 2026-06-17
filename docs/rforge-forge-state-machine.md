# `rforge forge` state machine

`rforge forge` is the guided workflow wrapper for the Meta-analysis spine. It must be resumable, auditable, and equivalent to lower-level CLI commands. The Go + HTMX cockpit may drive the same states, but it must not own scientific logic or bypass review gates.

## State model

Each state records:

- state ID and version;
- project path and project manifest reference;
- required inputs;
- produced artifacts;
- allowed transitions;
- blocking review gates;
- provenance event IDs;
- validation receipts;
- resume/replay hints.

State is stored in project-local workflow state, with durable provenance events for every transition.

## States

| State | Purpose | Required artifacts before exit | Blocking gates |
| --- | --- | --- | --- |
| `question_draft` | Capture research question and review intent | question text, review type, target outcomes/comparators if known | owner approves canonical question |
| `protocol_plan` | Convert question into protocol skeleton | inclusion/exclusion criteria, extraction schema seed, source plan draft | no auto-accepted LLM/keyword/entity suggestions |
| `source_plan` | Choose sources/connectors and dry-run queries | connector capability snapshot, query plan, privacy/auth/rate warnings | reviewer approves network/API plan |
| `import_plan` | Import source/reference records | import receipts, raw source refs, library diff | reviewer accepts import scope |
| `dedupe_review` | Resolve identity clusters | duplicate report, merge/split decisions, conflict records | unresolved identity conflicts block downstream package |
| `full_text_acquisition` | Queue and approve legal full-text candidates | OA/license metadata, candidate URLs, document asset plan | reviewer approves download/archive inclusion |
| `parser_arbitration` | Parse documents and resolve parser conflicts | parser manifests, parsed outputs, arbitration decisions | unresolved parser conflicts block accepted evidence |
| `indexing` | Build retrieval/domain-map indexes | retrieval lockfile, index receipts, benchmark/tuning config | embedding egress/privacy approval if remote models used |
| `screening` | Screen records through configured stages | screening decisions, conflicts, uncertain queues, active-learning runs | unresolved conflicts and required stages block analysis |
| `extraction` | Extract accepted evidence with source support | extraction schema, accepted evidence, correction history, gap report | unsupported accepted evidence blocks analysis/package |
| `analysis` | Prepare and run statistical models | input snapshot, scripts, outputs, warnings, checksums | analysis warnings must be acknowledged or resolved |
| `report_build` | Build report outputs and trace claims | report files, claim trace matrix, report audit | unsupported/weak claims block final export |
| `package_export` | Create Reproducible review package | package manifest, checksums, redaction report, replay command, audit report | audit failures block done state |
| `archive` | Move/package project safely | archive/restore receipt, private-state exclusions | path/privacy checks |
| `reopen_resume` | Resume or replay a project/package | state validation, lockfile validation, missing artifact report | incompatible versions require migration/decision |
| `done` | Package is audit/replay safe | passing package audit and replay receipts | none |

## Transition rules

```text
question_draft -> protocol_plan -> source_plan -> import_plan -> dedupe_review
  -> full_text_acquisition -> parser_arbitration -> indexing -> screening
  -> extraction -> analysis -> report_build -> package_export -> done
```

Allowed side transitions:

- any active state -> `reopen_resume` after process restart or project reopen;
- any active state -> prior state when a reviewer reopens a decision, preserving old decisions as superseded provenance;
- `package_export` -> `archive` for backup/share workflows;
- `archive` -> `reopen_resume` after restore;
- `analysis` -> `extraction` when analysis gaps require more evidence cleanup;
- `report_build` -> `analysis` when report audit exposes analysis artifact gaps;
- `parser_arbitration` -> `full_text_acquisition` when parser quality requires alternate full text.

Forbidden transitions:

- source/network execution before `source_plan` approval;
- document download/archive inclusion before legal acquisition approval;
- accepted evidence without source support;
- analysis without accepted evidence snapshot;
- final package export with unsupported claims, unresolved screening conflicts, or failed redaction/checksum audit.

## Review gates

| Gate | Applies at | Required reviewer decision |
| --- | --- | --- |
| protocol approval | `protocol_plan` | question, criteria, extraction schema seed are acceptable |
| network/API approval | `source_plan` | connector plan, credentials, rate limits, privacy risk accepted |
| identity approval | `dedupe_review` | merges/splits accepted or conflicts deferred with explicit status |
| legal acquisition approval | `full_text_acquisition` | download/archive/shareability decision recorded |
| parser arbitration approval | `parser_arbitration` | selected fields/passages/references accepted or corrected |
| screening approval | `screening` | required stages complete, conflicts adjudicated |
| evidence approval | `extraction` | accepted evidence has source support and correction history |
| analysis approval | `analysis` | warnings/method choices acknowledged |
| claim approval | `report_build` | claims trace to accepted evidence or are blocked |
| package approval | `package_export` | manifest, checksums, redaction, replay/audit pass |

## Provenance events

Each transition emits `forge.state.transition` with previous state, next state, actor, timestamp, inputs, outputs, blocking gates, validation receipts, and warnings.

State-specific events include:

- `protocol.plan.approved`
- `source.plan.approved`
- `library.imported`
- `identity.merge.approved` / `identity.split.approved`
- `document.acquisition.approved`
- `parser.arbitration.decided`
- `screening.decision.recorded`
- `evidence.item.accepted`
- `analysis.run.completed`
- `report.claim.audit.completed`
- `package.audit.completed`
- `package.replay.completed`

## CLI shape

Initial command family:

```sh
rforge forge init --project <path> --question <text>
rforge forge status --project <path>
rforge forge next --project <path>
rforge forge approve --project <path> --gate <gate> [--note <text>]
rforge forge reopen --project <path> --state <state> --reason <text>
rforge forge replay --project <path-or-package>
```

Lower-level commands remain authoritative and can advance state when they emit the same artifacts/provenance as `rforge forge`.

## HTMX cockpit mapping

- `/forge` shows state, blocked gates, next safe actions, CLI equivalents, and provenance timeline.
- Each state links to its workbench route (`/protocol`, `/sources`, `/dedupe`, `/acquisition`, `/parsing`, `/screening`, `/evidence`, `/analysis`, `/report`, `/package`).
- Buttons are disabled until required inputs are present.
- All forms degrade to normal HTTP posts and display the equivalent CLI command.

## Validation target

The first implementation should include a fake-backed CLI e2e that:

1. initializes a project in `question_draft`;
2. advances through at least `protocol_plan` and `source_plan` with approvals;
3. verifies forbidden transitions fail with actionable errors;
4. verifies `forge status` reports blocked gates and next actions;
5. verifies provenance events were emitted for every transition.
