# Meta-analysis spine acceptance-test matrix

This matrix maps each ordered Meta-analysis spine phase to validation coverage. Normal gates must remain local, deterministic, and fake/fixture-backed. Live services remain opt-in.

## Test layers

- **Unit**: package-level deterministic behavior.
- **CLI e2e**: `rforge` commands over temp Research projects.
- **Handler**: Go + HTMX handlers and view models.
- **Playwright**: opt-in browser paths behind `playwright_e2e` build tag.
- **Screenshot**: opt-in fixed-viewport regression where stable.
- **Provenance**: event-log and source-support assertions.
- **Replay**: Reproducible review package audit/replay checks.

## Matrix

| Phase | Unit | CLI e2e | Handler | Playwright | Screenshot | Provenance | Replay/package |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 0 Blueprint/gates | doc verifier for required headings/links | `todo-completion-audit` evidence check | n/a | n/a | n/a | n/a | n/a |
| 1 Question/source plan | query-plan compiler fixtures; connector capability validation | source-plan dry run over fake connectors | source-planning view model and dry-run partials | source plan approve/block path | source-plan overview optional | `protocol.plan.*`, `source.query.planned` | source plans included and dry-run replayable |
| 2 Import/identity/dedupe | importer fidelity, identity resolver, reversible decisions | import + dedupe + merge/split temp project | import/dedupe workbench handlers | dedupe review approve/split path | cluster table optional | `library.imported`, `identity.*` | source records and identity decisions included |
| 3 Legal full text | OA/license candidate selection, redaction rules | full-text candidate queue with fake OA sources | acquisition queue handlers | approve/skip acquisition path | acquisition table optional | `document.candidate.*`, `document.acquisition.approved` | only approved/shareable assets included |
| 4 Parser arbitration | parser manifest, field scoring, reference adjudication | fake parser comparison + arbitration | parser/reference workbench handlers | parser conflict resolution path | field comparison optional | `parser.run.completed`, `parser.arbitration.decided`, `reference.*` | parser manifests and selected outputs included |
| 5 Retrieval/graph/domain map | ranking, lockfiles, graph/table exports | rebuild/retrieve over fixture project | retrieval/graph/topic handlers | graph/table navigation path | graph/table screenshots | `retrieval.*`, `graph.*` | retrieval locks and graph artifacts included |
| 6 Screening/review | active-learning reproducibility, conflicts, stopping | multi-reviewer screening e2e | screening cockpit handlers | screen/uncertain/conflict/adjudicate path | screening dashboard optional | `screening.*` | screening audit, PRISMA counts, ranking runs included |
| 7 Evidence/gaps | source-support enforcement, gap analysis | extract/add/suggest/audit over fixture passages | evidence grid/gap handlers | evidence accept/correct/reject path | evidence grid optional | `evidence.*` | accepted evidence + schema + corrections included |
| 8 Statistics/methods | known-result effect fixtures, sensitivity/bias | prepare/run/export with fake runner; opt-in metafor | analysis workbench handlers | analysis warning acknowledgment path | analysis artifacts optional | `analysis.*` | input snapshot, scripts, outputs, warnings, checksums included |
| 9 Report/package | claim trace validator, package manifest/checksums/redaction | package create/audit/replay/restore e2e | package preview handlers | package preview/export blocked/unblocked path | package preview optional | `report.*`, `package.*` | full offline package replay required |
| 10 HTMX/forge | state reducer and next-action tests | `rforge forge init/status/next/approve/reopen` e2e | all cockpit route/partial handlers | complete happy path over fake project | key cockpit screens | `forge.state.transition` | package generated from cockpit path replayable |
| 11 Broader cockpit | graph/timeline/benchmark units | optional after package safety | broader cockpit handlers | optional exploratory paths | optional | provenance journal events | benchmarks and roadmap reports included when generated |

## Required closeout commands

Before checking off a phase implementation slice, run the smallest relevant subset plus the normal gate where practical:

```sh
go test ./...
go vet ./...
make todo-completion-audit
go run ./cmd/rforge oss inventory-check opensource/inventory/manifest.json
git diff --check
```

Browser slices additionally require:

```sh
make web-gui-smoke
RFORGE_RUN_PLAYWRIGHT=1 go test -tags playwright_e2e ./internal/webui -run TestPlaywright -count=1 -v
```

Live-service slices must keep live checks opt-in and document environment variables.

## Phase 0 acceptance

Phase 0 is complete when these planning artifacts exist and are referenced from `TODO.md`:

- `docs/meta-analysis-spine-blueprint.md`
- `docs/meta-analysis-spine-roadmap.md`
- `docs/rforge-forge-state-machine.md`
- `docs/reproducible-review-package.md`
- `docs/meta-analysis-spine-acceptance-matrix.md`

and `make todo-completion-audit` passes.
