# Live scouting E2E follow-up

Status: Accepted

## Working brief and acceptance authority

Researchers and host agents should be able to resume a metadata search, understand its outstanding screening work, and build an explicitly unscreened bibliography without acquiring full text. The user requested fixes and improvements to rforge from the phononic-thermal-diode E2E. In the subsequent LGTM checkpoint, the user accepted the specific recommendation to support bibliographies of discovered papers before screening/download, labeled as unscreened rather than evidence of inclusion or full-text review. This is a standalone planning handoff, not implementation, installation, live API, service-control, push or release authority.

**Planner's calls:** preserve the existing downloaded-paper default; add explicit `--scope discovered` and `--scope downloaded` selection to `citations build`. Discovered mode uses local search metadata without network enrichment. Make the resume text truthful with the smallest presentation change, and add queue-relevant screening counters without redefining legacy counts. No additional product decision remains; new unrelated features are excluded.

## Evidence and current state

Discovery baseline: `db7da10e2eea22f08122d493a9f4c33e9deaf2a8`. The tracked worktree/index are clean; unrelated untracked research job state remains untouched. Task store preparation reports READY, 87 records, ten goals, no drafts and no open tasks. Runtime/admission ownership is not assessed here; no worker is dispatched by this plan.

**OBSERVED, manual-run receipt:** `e2e-phononic-thermal-diodes/report.md` in the user's external research directory records installed binary `0149ef1`, 261 raw / 206 deduped records, seven preserved arXiv failures, no-new-request resume at its total cap, 257 actionable queue rows versus 261 raw pending, and zero downloaded-paper citations. Those are software-run observations, not scientific quality or adoption measurements. Future acceptance uses synthetic offline fixtures, not that mutable external directory or live endpoints.

**Already delivered:** [V0-033](../../tasks/items/V0-033-0eda4436.md) and its [receipt](../../tasks/evidence/V0-033-0eda4436.toml) show the malformed-report repair merged in `5d6297f`; current source contains it. The installed binary remains older. Do not duplicate that repair or treat old-binary failures as a new regression. Reinstallation/retesting is a separate later execution operation, not a new product-code task or hidden planner action.

| Goal → accepted interface → source observation → gap / disposition |
| --- |
| Truthful resume → [budget contract](../cli.md#search-budgets-and-stop-reasons) → `internal/search/batch.go:112–134` can stop at an exhausted cap before any failed pair is requested; `internal/cli/source_commands.go:1570–1572` unconditionally says failed pairs were retried → presentation-only resume task below. |
| Understand screening work → directory queue/progress → `screen_dir_command.go:75–98` skips no-ID/decided rows; `:253–287` counts raw records and log entries → add actionable queue and missing-identifier counters; preserve legacy fields. |
| Bibliography before full text → user-approved unscreened discovery use case and README quickstart → `citations_command.go:69,193–251` requires a matching PDF slug; `:148–151` describes downloaded papers → explicit local discovered scope, with old default and wording preserved for downloaded mode. |
| Scientific and delivery proof → existing release/benchmark goals → no new scientific labels, live qualifications or package approvals supplied → remain unmet/deferred, not replaced by these fixture checks. |

The derived engineering goal **RF-SCOUTING** is trustworthy metadata-only scouting outputs: honest recovery messages, explicit screening count domains and clearly unscreened discovery bibliographies. The new goal is scoped to these three outcomes, not ResearchForge-wide quality or a new research strategy. Existing goals, completed receipts and the [RF-BENCH owner pause](rf-bench-offline-round-2.md#owner-decision-planning-remains-paused) stay unchanged.

## Decisions and executable specifications

### Resume summary

Remove the unconditional claim that failed pairs were retried. Use the neutral text `resumed from checkpoint` plus the existing stop reason, request usage, coverage and saved-output summary. Do not report the traversal-local skipped counter as total corpus recovery, add a new accounting model, reset budget, change `resumedPairs`/checkpoint JSON, change exit codes or retry behavior. The source already enforces the cap; this is the smallest complete correction to what the human sees.

Proposed `TestSearchBatchResumeSummaryTruthful` uses local HTTP fixtures: a persisted failed pair with exhausted cap produces zero additional server calls and no retry-success claim; a raised total cap actually retries and preserves the ordinary completion/remaining-failure behavior. Reuse `TestSearchBatchFailedResumeKeepsUsageAndFailures`, `TestSearchBatchResumePreservesRequestBudget` and the checkpoint tests rather than add a new harness or live test.

### Queue-relevant screening progress

Keep JSON `total`, `decided`, `included`, `excluded`, `uncertain`, `pending` and their current count domains unchanged. Add:

- `queue_pending`: exactly the number of rows a fresh `screen queue` would emit for the same inputs, including repeated raw records when the current queue includes them.
- `missing_identifier`: count of raw records whose trimmed DOI and arXiv ID are both empty. Such rows are not actionable by the directory queue.

Text output retains its old two lines and adds an explicitly labeled line with these two counters. Reuse the existing DOI-first/arXiv-fallback key and decided-set rules from `screenDirLoadDecided`, preferably factoring a tiny same-file helper over already loaded entries rather than reading the log twice or inventing a new state store. Do not dedupe raw rows, reinterpret decisions, change screening approval, mutate logs, or repair unrelated historical decision semantics.

Proposed `TestScreenDirProgressMatchesQueueEligibility` creates titled PMID-only or OpenAlex-only records (valid scholarly identifiers, but no DOI/arXiv), DOI/arXiv records, duplicate raw identifiers and recorded decisions; compare `queue_pending` against actual CSV row count and verify `missing_identifier`, text/JSON output, unchanged legacy counters and byte-identical inputs. The shared `readResultsJSONL` requires a title and at least one scholarly identifier (`source_commands.go:989–1012`); title-only records are invalid and must remain rejected. Existing malformed results/log, BOM/CRLF, unknown-decision import and symlink guards remain required.

### Explicit unscreened discovered-paper bibliography

Extend `rforge citations build --research-dir <dir> [--out <file>]` with `--scope downloaded|discovered`, default **downloaded**. Omitted/explicit downloaded scope preserves existing PDF selection, enrichment, sorting/formatting, warning handling, text/JSON shape and publication behavior. Validate missing or unknown scope values as usage failure (exit 2, `usage`) before any input read, network call or output change.

Discovered scope:

1. Traverse the same immediate topic directories and strictly read their existing `results.jsonl` using `readResultsJSONL`; do not require PDFs, screening logs, a project or new manifest. Skip records without DOI/arXiv ID, as the existing collector does. Deduplicate by the existing trimmed DOI-first/arXiv-fallback exact key across topics, retain the first encountered record in deterministic topic/file order, and keep current identifier normalization rules unchanged. Do not merge fields, conflate DOI/arXiv aliases or turn this into another deduplication engine.
2. Format citations from saved title, authors, year, venue and identifiers through existing citation-formatting helpers. Leave missing metadata absent; do not fabricate it. Do not call Crossref, arXiv or any other network service in discovered mode. Reuse existing formatting/transaction code instead of introducing dependencies, a new export command, BibTeX/CSL formats or an enrichment option.
3. Markdown prominently states **Unscreened search metadata — not evidence of inclusion or full-text review**. JSON discovered-mode results add `scope: "discovered"` and `screening_status: "not_assessed"`; retain the current citation entries/count/warnings shape. `--json` keeps its current stdout-only contract (no Markdown file write even with `--out`). Discovered selection does not consult screening decisions: it must not imply records were approved, included or reviewed.
4. Preserve filetxn-based replacement, readable discovery errors, malformed-row fail-closed behavior, symlink protection and no partial output on discovery failure. Do not read PDF directories in discovered mode. Normal Markdown mode saves its requested/default output exactly as the old command does. Empty discovery yields an explicitly unscreened zero-citation artifact, not an acquisition recommendation.
5. Update both command/help registrations in `internal/cli/cli.go`, `docs/cli.md` and README's bibliography quickstart. The quickstart must explicitly request discovered scope and identify its unscreened meaning; document the compatible downloaded default and no-network discovered behavior. Do not claim all legacy/public skill examples have changed or edit executable skill files as planning collateral.

Proposed `TestCitationsBuildDiscoveredScope` covers no-PDF multi-topic inputs, duplicate identifiers, local authors/year/venue and sparse metadata, an explicitly failing/instrumented network transport (zero calls), unscreened text/JSON labels, valid empty inputs and default downloaded compatibility. Proposed `TestCitationsBuildScopeValidation` covers missing/invalid values and unchanged output/no side effects. Extend existing malformed/symlink/transaction guards for discovered scope; preserve every downloaded-paper test. A missing new test or locally constructible fixture is a deliverable, not an external blocker.

## Batch, ordering and exact checks

One complete batch, three independent Now tasks under RF-SCOUTING; no product prerequisites, no rolling planning successor. Rank the two observed misleading-output repairs first, then the approved bibliography capability. A single task owns all bibliography registry/help/README edits. Its actual shared resources are `cli-surface` and `agent-docs`; the other two tasks touch neither resource. Expected write sets are disjoint; three lanes may be independently eligible, but this is not observed runtime/admission authority or a promise of parallel dispatch.

| Lane | Expected writes | Proposed/unrun owning check |
| --- | --- | --- |
| Resume summary | `internal/cli/source_commands.go`, `internal/cli/search_budget_test.go` | `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/cli -run 'TestSearchBatchResumeSummaryTruthful|TestSearchBatchFailedResumeKeepsUsageAndFailures|TestSearchBatchResumePreservesRequestBudget|TestExecuteSearchBatchResume' -count=1` |
| Screening counters | `internal/cli/screen_dir_command.go`, `internal/cli/screen_dir_command_test.go` | `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/cli -run 'TestScreenDir' -count=1` |
| Discovered bibliography | `internal/cli/citations_command.go`, `internal/cli/citations_command_test.go`, `internal/cli/cli.go`, `docs/cli.md`, `README.md` | `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/cli ./internal/docs -run 'TestCitationsBuild|TestHelp|TestExecuteHelp|TestCommand' -count=1` |

Every task starts red with its named behavior test, implements the smallest correction, refactors within its seam and receives normal independent implementation review. Required guard is `make check` under the repository's normal worker/merged-tree contract; affected-only checks are not a full-main pass. Bounded owning-path prerequisite repairs required by those checks are permitted without weakening assertions. No live APIs in normal tests, no new dependencies, no shared task-store edits by workers.

## Invariants, pre-mortem and stopping rule

- Incorrectly broadening a generic approval could change all bibliography defaults: preserve downloaded mode; user approved the unscreened capability, while explicit scope/no-network are documented planner calls.
- New progress counters could silently change old clients' meanings or drift from queue rules: additive fields, exact emitted-row comparison and legacy-value assertions; no queue mutation.
- Removing misleading wording could accidentally reset/retry budget: only summary presentation changes, local-server request counts and existing checkpoint guards remain binding.

Validation measures observable behavior, not invented ROI, quality, coverage or speed percentages. No full-text acquisition, screening/extraction approval, final scientific claims, package export, external credentials or Autowork journal repairs are in this batch. Finish after saving/validating/reviewing/committing these planning records. New tests above are proposed/unrun during planning. Installation of current source and rerunning saved E2E controls remain separate later operational work; no duplicate corruption-fix task is created.

## Saved handoff and builder dispositions

- [V0-034-171e78aa](../../tasks/items/V0-034-171e78aa.md): Now rank 1000, truthful resume summary. Next action: the failing `TestSearchBatchResumeSummaryTruthful`.
- [V0-035-4e5aa188](../../tasks/items/V0-035-4e5aa188.md): Now rank 2000, actionable screening counters. Next action: the failing `TestScreenDirProgressMatchesQueueEligibility`.
- [V0-036-a1fd3a5e](../../tasks/items/V0-036-a1fd3a5e.md): Now rank 3000, approved unscreened bibliography. Next action: the failing `TestCitationsBuildDiscoveredScope`.
- [RF-SCOUTING](../../tasks/goals/RF-SCOUTING.md): priority 27, unmet, no focus change. All three lanes have no product dependencies. Task V0-036 alone reserves cli-surface and agent-docs.

**Independent builder review:** no consequential design question remains for the resume or bibliography lanes. It confirmed the disjoint write sets, real resource constraints, downloaded-default/no-network/JSON boundaries and locally reachable checks. **Planner-resolved fixture finding (V0-035):** a title-only record cannot reach the progress counter because the shared results reader rejects it. Use a titled PMID-only/OpenAlex-only record instead; it passes existing validation and still lacks DOI/arXiv for queue purposes. Do not weaken/bypass shared validation or broaden V0-035's writes to that reader. The corrected fixture and this disposition are saved in its task record. Internal helper organization remains worker-reversible; no owner-only blocker remains.

Planning validation covers final task-store/readback/link/whitespace consistency and the existing pre-commit full check. Exact results are reported at handoff; new product tests remain proposed/unrun. Local selection is checked only after the planning commit; runtime ownership/capacity remains unassessed. Existing research artifacts, completed tasks and owner pauses must remain untouched.
