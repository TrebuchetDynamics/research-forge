# RF-CLAIMS offline proof: stored-evidence facets

This is the aggregate evidence for CLAIMS-001-82, not a declaration that a
scientific conclusion is true. Start with the [accepted plan's acceptance
criteria](../verifiable-engine-plan.md#acceptance),
[ADR 0009](../adr/0009-verification-facets-and-provenance-levels.md) and the
[schema's deterministic checks](../claim-evidence-schema.md#deterministic-checks).
The original contracts are [V-D1](../../TODO.archive.md#v-d1),
[V-D2](../../TODO.archive.md#v-d2), [V-D3](../../TODO.archive.md#v-d3) and
[V-D4](../../TODO.archive.md#v-d4).

## Qualification and revision boundary

- **Executed focused proof:** all nine claim-related packages passed at
  `5cadf3b21256505028b39a4dea982afba03b2999` (the worker baseline), tree
  `479e2184ff8a9dec38066124af9a8e02ef15493e`, using Go 1.26.6 on linux/amd64.
- **Verified historical aggregate proof:** `make check` passed on the V-D4
  integration commit `68292c5edbad9ad475ccc299a6a1be232daf7e12`, tree
  `8a9f1a8acafbded08587113de1db60c25e024ce9`. This revision contains V-D1
  through V-D4 together; the result is not inferred from their task closure.
- The only paths changed between those revisions are `TODO.archive.md`,
  `TODO.md` and `goals.json`. Code, tests, build inputs and `Makefile` are
  unchanged. Nevertheless, the historical result is **not** a full-check pass
  for the baseline's different tree, particularly its changed tracking inputs.
- This task adds documentation only, without changing behavior or tests.
  RF-CLAIMS remains **unqualified in the goal registry** until normal closeout
  records executed passing evidence through `goals.py evidence`. No goal or
  task registry is changed here. The runner's merged-tree full gate remains
  required; it is not replaced by this document.

## Commands and results

### Focused check executed in this task

Working directory: repository root, clean baseline revision named above.
The live-test opt-in variables were unset; fixtures and mock transports were
used, not live services. Exact command (exit **0**):

```sh
env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/parsing ./internal/documents ./internal/claims ./internal/cli ./internal/evidence ./internal/report ./internal/benchmarks ./internal/reviewpkg ./internal/mcp
```

All nine packages returned `ok`, with no cached package results in this run:

| Package (`internal/`) | Result | Elapsed reported by Go |
| --- | --- | --- |
| parsing | pass | 0.068s |
| documents | pass | 0.013s |
| claims | pass | 0.012s |
| cli | pass | 33.456s |
| evidence | pass | 0.005s |
| report | pass | 0.004s |
| benchmarks | pass | 0.120s |
| reviewpkg | pass | 0.104s |
| mcp | pass | 0.110s |

Raw output: task-local `$TMPDIR/rf-claims-proof/focused.log`, SHA-256
`b87b73090a36d61dd322897660f508762033c553cdaeb263fda6412210641c81`.
That scratch artifact is not a permanent repository dependency; the executed
command and complete package summary are retained above.

### Existing full check verified, not rerun

The worker instructions prohibit running the full check locally. The existing
runner artifacts were inspected instead. They bind the command **`make check`**
to integration tree `8a9f1a8acafbded08587113de1db60c25e024ce9`:

| Artifact | Binding/result | SHA-256 |
| --- | --- | --- |
| Common Git dir: `autowork-execution/7130d992ef244c70a5c015d04cc8c4a8.json` | `outcome: EXECUTED`, verification wrapper exit 0, `context.integration_commit: 68292c5edbad9ad475ccc299a6a1be232daf7e12` | `768ae553f4433c5358d73f6079e1ebb38d09694d195cb959aa316d4eda59499c` |
| Runner state: `full-check/tree-8a9f1a8acafbded08587113de1db60c25e024ce9.json` | `command: make check`, `ok: true`, empty failure reason, task `V-D4` | `272cf80746a6c205f6f5f4ee903c62d4229a50ac080ecca91cdcad2e627ee40d` |
| Runner state: `full-check/tree-8a9f1a8acafbded08587113de1db60c25e024ce9.log` | Complete full-check output, including final `git diff --check` | `33be45e3150d8890bdcc65b71983f16c04377b65e0310ed90181d5275012ccba` |

“Common Git dir” is resolved with `git rev-parse --git-common-dir`; the inspected
runner-state directory is `$HOME/.local/state/pi-autowork/d0648c07b9fe8e30`.
These are existing execution artifacts, not a new receipt or registry.
The wrapper's own `.log` is empty and is **not** used as test evidence.
The tree-keyed log shows every [Makefile](../../Makefile) check stage:

1. Formatting: `test -z ""` (no files reported by gofmt).
2. Module consistency: `go mod tidy -diff`.
3. `go test ./...`: **33 packages `ok`**, including all nine above; `cmd/rforge`
   has no test files. Existing Go cache hits are visible for some other packages.
4. `go vet ./...`.
5. `go run ./cmd/rforge --json decisions`: empty decision list, `ok: true`.
6. `go run ./cmd/rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md`:
   completion audit verified, zero blocking owner decisions.
7. `go run ./cmd/rforge oss inventory-check opensource/inventory/manifest.json`:
   `inventory ok: 23 entries`.
8. `git diff --check`.

The verified historical command is recorded as written; the artifacts do not
record an `env -u` prefix. The task's exact sanitized full-check command below
was **not executed by this worker** and has no claimed exit result on the current
tree. It is left to the normal runner gate, not listed as a live check or a
fabricated pass:

```sh
env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE make check
```

## Contract-to-test coverage

These are existing cases inspected at the focused-check revision and included in
that passing package run. Spaces in subtest names appear as underscores in Go
output. Passing tests establish only the fixture properties described here.

| Contract | Existing test seam and case | What it establishes |
| --- | --- | --- |
| V-D1: stable identity and hash locality | [`internal/parsing/integrity_test.go:56`](../../internal/parsing/integrity_test.go), `TestS2ORCReparsePreservesPassageIntegrity/{unchanged,normalized_whitespace,one_character_edit}` | Unchanged normalized content retains passage IDs and document/passage hashes across parser versions; a one-character edit changes the document hash and only the affected passage hash. |
| V-D1: normalized document digest | [`internal/parsing/integrity_test.go:27`](../../internal/parsing/integrity_test.go), `TestEnrichParsedDocumentHashesTextStream` | SHA-256 covers the declared text stream; non-text metadata does not change it, while title/reference edits do. `internal/documents` also passed; no new document behavior is claimed. |
| V-D2: append-only storage and attribution | [`internal/cli/claim_command_test.go:17`](../../internal/cli/claim_command_test.go), `TestClaimCommandsAppendAndReadSnapshots` | `claim add/list/show` preserve prior snapshots and select the latest claim; semantic support remains `agent_reported`, named approval is retained, mechanical facets are not invented. |
| V-D2: reject unauthorized attribution | [`internal/cli/claim_command_test.go:133`](../../internal/cli/claim_command_test.go), `TestClaimAddRejectsMisattributedFacets/{engine_semantic_support,missing_agent,missing_model,unnamed_reviewer}` | Invalid facet input fails at the CLI seam without touching the ledger. |
| V-D3: independent deterministic facets | [`internal/claims/check_test.go:14`](../../internal/claims/check_test.go), `TestCheckDeterministicFacets/absent_quote`; [`internal/cli/claim_command_test.go:84`](../../internal/cli/claim_command_test.go), `TestClaimCheckPersistsSeparateFacets` | Source identity and content integrity pass while passage match fails for an invented quote; semantic support and human approval are not inferred. CLI persists the separate outcomes by appending a snapshot. |
| V-D3: actual quote matching and trace | [`internal/claims/check_test.go:94`](../../internal/claims/check_test.go), `TestMatchPassageOriginalByteOffsets`; [`internal/evidence/citation_passage_match_test.go:10`](../../internal/evidence/citation_passage_match_test.go), `TestCitationLocksCheckStoredText`; [`internal/report/citation_trace_test.go:13`](../../internal/report/citation_trace_test.go), `TestBuildCitationEvidenceTraceViewLinksClaimsToFullProvenanceChain` | Exact then canonical Unicode/whitespace matching returns original byte offsets, rejects absent text, and exposes quote failures in the citation trace even with complete provenance links. |
| V-D3: citation-correctness metric | [`internal/benchmarks/citation_correctness_test.go:12`](../../internal/benchmarks/citation_correctness_test.go), `TestScoreCitationsChecksEveryOccurrence` | Recomputes instead of trusting supplied passing facets; duplicate, absent, missing and unresolved citations all enter the denominator (fixture 2/5). Empty input is `no_citations`, not success. |
| V-D4: facet-based final export guard | [`internal/report/claim_panel_test.go:14`](../../internal/report/claim_panel_test.go), `TestFinalExportReadsSeparateFacets` (including `passageMatch/failed`); [`internal/cli/report_trace_test.go:18`](../../internal/cli/report_trace_test.go), `TestExecuteReportTraceWritesCitationEvidenceTraceView` | Failed/inconclusive recorded facets block export even if a panel says ready. Checked ledger facets reach report guards; changed text, quote or source bindings require reassessment. |
| V-D4: package audit | [`internal/reviewpkg/audit_test.go:221`](../../internal/reviewpkg/audit_test.go), `TestAuditRejectsUnsupportedIncludedReportClaim/{failed_passage_facet,mismatched_quote}` | A package with otherwise complete trace links is rejected for a failed passage facet or mismatched quote. |
| V-D4: MCP/CLI parity, agent support, no approval tool | [`internal/mcp/server_test.go:31`](../../internal/mcp/server_test.go), `TestToolCLIParity`, especially `claim_check` | Tool list is exactly seven tools, including `claim_check` and no approval tool. MCP accepts named agent/model semantic support as `agent_reported`; CLI and MCP JSON agree after removing only the three engine assessment timestamps. Engine semantic support, missing model and `human_approval` submissions are rejected without appending snapshots. |

## Limits and closeout

The V-D1 (`7a5430a`), V-D2 (`20ab2da`), V-D3 (`43f3faf`) and V-D4
(`cc97226`) worker reports at common-Git-dir
`autowork-execution/slot-data/<task>/attempt-1/report.json` were inspected:
they report passing focused checks and no pending live checks. They supply
history, not independent aggregate proof; the tree-bound full-check artifacts
above supply the historical aggregate result.

There are no live-service, physical-device or human-review results claimed by
this proof. Attribution guards do not authenticate an agent or reviewer, and
neither mechanical checks nor semantic/human facets establish scientific truth.
No approval tool, product scope, tests or implementation are added here.

For closeout, use this document plus the runner's executed merged-tree gate for
normal goal evidence. A missing or failed current full check remains unqualified.
Revisit this proof whenever the parser/hash representation, claim schema/store,
checker, export/audit guards, MCP mapping, tests or build inputs change; old
results remain bound to their recorded revisions.
