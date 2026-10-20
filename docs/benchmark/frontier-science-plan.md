# Frontier-science stress-test plan

Status: Accepted

This is a planning-only handoff for finding rforge bugs and improvement opportunities with difficult research topics. The owner requested a couple of topic-based tests and delegated planning through Plan For Me. Topic selection and operating defaults below are **planner's calls**, not additional owner decisions. No research runs, implementation, spend, commit or publication occurred during planning.

Tasks: [V-FS1 and V-FS2](../../TODO.md#now). This exploratory slice supports RF-BENCH but does not change its held-out-ground-truth definition, existing task IDs or [release gate](../verifiable-engine-plan.md#acceptance).

## Brief and boundaries

**Size:** small project: one offline preparation task, then one separately gated live campaign.

**Goal:** exercise the actual search → dedupe → paper resolution/citation expansion → evidence-led draft → provenance-validation journey on two contrasting topics; produce reproducible defect candidates and ranked improvement opportunities rather than just a bibliography.

**Success:** an executable two-case packet; later, one auditable dossier per topic and a cross-topic findings file, with exact commands, failures, source support, uncertainty and evidence boundaries. A run finding no reproducible engine defect is valid. Failed or gated stages remain visible, not silently skipped.

**Non-goals:** implement fixes during a baseline run; build a general benchmark framework; execute the 48-run Pi pilot; add connectors, models or services; run a pooled meta-analysis; acquire PDFs, import screening approvals, accept suggestions/claims or export packages without the existing human gates; commit or publish research artifacts.

**Anti-goals:** fabricated measurements or citations; experimental signatures presented as conclusive proof without support; pressure/material/measurement conditions collapsed into one number; provider outages presented as negative scientific evidence; a mechanical provenance pass presented as scientific approval; frontier runs promoted to gold sets, recall estimates or release evidence.

**Invariant for every task:** production behavior, existing locked benchmark inputs and immutable engine receipts stay unchanged; normal tests remain offline; host settings and unrelated dirty work are untouched.

## Evidence and approach

Repository inspection at HEAD `a743265c23964d747f7970787a4ddc4360f5aa40`, including current dirty source (not just HEAD):

- [Existing plan](../verifiable-engine-plan.md#owner-answers-2026-10-19-follow-up), [benchmark spec](spec.md), [agent-tier runbook](agent-tier.md) and TODO V-B6-run already own the eight-question × two-condition × three-repetition pilot, held-out leakage controls and live authorization. Replacing these with frontier questions would invalidate comparability.
- [Repository usage skill](../../skills/research-forge/SKILL.md#search-depth-and-stopping) requires Standard by default, explicit stop/budget accounting and schema-v2 provenance; user-requested Comprehensive is allowed. The separately installed user skill still recommends Comprehensive by default; this repository's current instructions govern this work. Do not edit either skill as part of this campaign.
- [DepthBudget](../../internal/search/batch.go) gives Standard 10 queries / 5 sources / 50 connector calls, Comprehensive 30 / 20 / 600. One budget unit is a connector Search invocation, **not necessarily one HTTP attempt**; retries and citation/paper calls need separate accounting.
- [Source presets and CLI](../../internal/cli/source_commands.go) define `scholarly-fast` as OpenAlex, Crossref, Semantic Scholar and arXiv; API clients can retry. `all` includes many irrelevant or credential-gated sources. Use the four-source preset, record each failure, and do not wait indefinitely on one provider.
- Existing offline guards include `internal/cli/search_budget_test.go::TestSearchBatchResumePreservesRequestBudget`, `internal/cli/search_budget_test.go::TestSearchBatchBudgetStops`, `internal/cli/search_resume_dryrun_test.go::TestExecuteSearchResumeDryRunListsPendingFailuresWithoutNetwork` and `internal/cli/provenance_validate_test.go::TestProvenanceValidateAcceptsConformingFile`. Reuse these instead of duplicating the search engine or inventing a capture system.
- [Provenance facets](../adr/0009-verification-facets-and-provenance-levels.md), [claim schema](../claim-evidence-schema.md) and [contribution rules](../../CONTRIBUTING.md#research-quality-improvement-claims) distinguish mechanical evidence properties, agent semantic judgment and human approval. Frontier artifacts are exploratory/development material, never held-out tuning data.
- README, installer, several CLI/source files and `research/project-success-2026-10/` were already dirty/untracked. Preserve them; the existing project-success research is not this campaign's baseline. Build the tested binary from an explicitly recorded snapshot at execution time, not silently from the installed release.

### Options compared

1. **Chosen: two exploratory dossiers over existing CLI/services.** Fast route to observable bugs; supports contested/heterogeneous literature without pretending to know total relevant studies.
2. **Run the existing paired 48-run pilot now.** Rejected for this slice: different purpose, gold-set prerequisites and live qualification/spend gates. Leave its accepted design intact.
3. **Build a new scored benchmark across all twelve topics.** Deferred: increases cost and framework work before the two-topic trial establishes useful failure modes; no defensible gold inclusion sets are provided.

No new ADR is needed: these choices are reversible and do not change production architecture.

## Planner's calls

- **Initial pair/order:** Majorana zero modes, then nickelate superconductivity. The first targets signature false positives; the second targets quantitative comparability. Emergent spacetime, Hubble tension, black-hole information and Gaussian primes remain proposed later topics, not queued work.
- **Unit of work:** one host-agent-assisted CLI research run per topic using the same recorded binary and agent/model identity. No without-rforge condition, repetitions, statistical effect estimates or claim of tool superiority.
- **Depth:** Majorana Comprehensive because the supplied first-test prompt explicitly asks for it; nickelates Standard initially. Query counts are upper plans, not completion quotas. Stop on the engine's actual saturation/error/budget signal.
- **Search defaults:** four `scholarly-fast` sources, limit 20 per query/source; Majorana at most 100 connector search calls total; nickelates at most 50. No automatic increase. A needed nickelate Comprehensive escalation is recorded as an unfinished follow-up, requiring a new allowance before more calls. These are proposed live ceilings, not permission to spend.
- **Additional live bounds:** at most 3 retrieved citation seeds per topic, depth 1, `--max-records 30 --max-api-calls 6 --retry-budget 0` per expansion; at most 5 `paper` resolutions per topic. Do not count these as part of `search batch`'s request cap. All network commands get a 120-second process timeout, preserving partial output/checkpoints; at most one resume per search within the original total allowance, never an unbounded retry loop. Report HTTP attempts only when observable; otherwise mark unknown.
- **Credentials/cost:** existing authorized provider access only; no paid account setup, new subscriptions, configuration changes or purchases. V-FS2 stays in Needs decision until an operator authorizes the live scope, agent/token/time/spend budget and any needed human review. Runtime auth/quotas are checked then, not fabricated in planning.
- **Artifacts:** prepare `docs/benchmark/frontier-science/`; live results go only under `research/frontier-science/<campaign-id>/`. Refuse to overwrite an existing campaign root. Record UTC start, binary SHA-256/version, Git revision plus dirty status/diff fingerprint, OS, agent/model, budgets and commands. Never copy `.env`, credentials, full session secrets or unrestricted copyrighted text.
- **Scientific status:** all drafts and semantic comparisons are `agent_reported`; use `unknown`/null with a reason where source text, uncertainty or replication is unavailable. Corrections, retractions and preprint/journal version links are checked from primary records when accessible, otherwise recorded as gaps. No self-approved scientific conclusions.
- **Scoring:** use the five suggested dimensions as an audit checklist, with observed counts and source-linked explanations where meaningful. The proposed 30/25/20/15/10 weights remain a proposal, not an existing rforge metric or a weighted pass mark. Retrieval recall and mathematical/statistical correctness stay not-assessed where an independent denominator or derivation is absent; do not renormalize missing scores into a flattering total.
- **Bounded iterations:** two initial topic runs, at most one within-cap resume each; no implementation/fix loop. Triage first; plan individual red-green-refactor fixes only after a reproducible failure is established.

## Two case cards

The preparation task transcribes these questions, query plans and evidence requirements into `cases.json` and the runbook. Query wording is a starting plan, not a scientific finding. Do not treat searches as Boolean syntax supported identically by every provider; preserve actual submitted queries.

### Majorana

**Question:** What experimental evidence is sufficient to establish topological Majorana zero modes in semiconductor–superconductor hybrid nanowires, and which published signatures can also arise from trivial Andreev bound states?

Twenty planned variants (stop earlier if warranted):

1. Majorana zero modes semiconductor superconductor nanowire experimental evidence
2. Majorana nanowire zero bias peak trivial Andreev bound states
3. Majorana nanowire quantized conductance replication
4. Majorana nanowire quantized conductance correction retraction
5. Majorana nanowire disorder induced zero bias peaks
6. Majorana nanowire smooth confinement Andreev states
7. Majorana nanowire nonlocal conductance
8. Majorana nanowire bulk gap closing reopening
9. Majorana nanowire topological gap protocol
10. Majorana nanowire topological gap protocol criticism
11. Majorana nanowire Coulomb blockade parity
12. Majorana nanowire parity lifetime poisoning
13. Majorana nanowire conductance temperature dependence
14. Majorana nanowire magnetic field angle dependence
15. Majorana nanowire length dependence energy splitting
16. Majorana nanowire Josephson fractional periodicity trivial explanation
17. Majorana nanowire braiding non Abelian experimental evidence
18. Majorana nanowire correlated end states
19. Majorana nanowire independent replication experimental signatures
20. Majorana nanowire review experimental limitations

**Outputs:** reproducible search record; contradictory-evidence matrix; signature comparison; exactly five **proposed**, falsifiable predictions with observations that would refute them and source rationale; uncertainty assessment; source-backed draft verdict distinguishing demonstrated results, model-dependent interpretation and open questions. If five supported proposals cannot be formulated, retain explicit unsupported slots with reasons rather than inventing predictions.

**Matrix columns:** claim/signature; study identifier and version/status; experimental vs theoretical; device/material, field, temperature and measurement conditions when reported; observed result with quote/location; topological interpretation; trivial alternative; independent replication status; uncertainty; source locator; agent-vs-engine verification facets. Distinguish replication by an independent group from multiple papers on one device/dataset.

### Nickelates

**Question:** Which pairing mechanisms and structural conditions explain superconductivity across nickelate families, and which experimental results are genuinely comparable?

Six planned variants:

1. nickelate superconductivity infinite layer thin film substrate oxygen stoichiometry
2. nickelate superconductivity bilayer high pressure transition temperature
3. nickelate superconductivity ambient pressure structural phase
4. nickelate superconductivity resistive onset zero resistance magnetic shielding
5. nickelate superconductivity independent replication pressure calibration
6. nickelate superconductivity pairing mechanism experimental constraints review

**Outputs:** reproducible search record; contradictory-evidence/conditions matrix; signature comparison; uncertainty assessment and source-backed draft verdict. No pooled transition-temperature estimate or mechanism selection from incompatible measurements.

**Matrix columns:** study/version/status; family/composition; structure/phase; film vs bulk, substrate and sample preparation; pressure value/unit and calibration; transition temperature value/unit **and definition** (onset/midpoint/zero resistance/magnetic); measurement method; reported uncertainty or null; sample/dataset/group provenance; replication status; mechanism claim vs direct observation; quote and source location. Preserve original units alongside explicit conversions; do not infer missing pressures, uncertainties or shielding fractions.

## Executable preparation and live handoff

### V-FS1 — Offline packet and mechanical artifact checks

Expected writes (four files): `docs/benchmark/frontier-science/cases.json`, `docs/benchmark/frontier-science/runbook.md`, `docs/benchmark/frontier-science/check.py`, `docs/benchmark/frontier-science/test_check.py`. No production changes or new dependencies. Python standard library only, no general framework or scoring engine.

Implement a small read-only CLI:

`python3 docs/benchmark/frontier-science/check.py --cases docs/benchmark/frontier-science/cases.json [--run research/frontier-science/<campaign-id>]`

Exit 0 = structurally complete packet/dossier; exit 1 = missing, malformed, unsafe or inconsistent artifact. Print precise errors. It does **not** adjudicate scientific support, prove a claim, grant approval or call the network.

`cases.json` has `schema_version: 1`, `purpose: "exploratory_not_gold"`, ordered `cases` with `id`, `question`, `depth`, `sources` (four explicit names), `queries`, `max_search_calls`, `limit`, `required_outputs` and `evidence_columns`. IDs are `majorana` and `nickelates`. Require the selected order, nonempty questions/queries, cap-consistent plans and the scientific/evidence requirements above. Reject duplicate IDs, unrecognized depths, path traversal and any claim that these are gold sets.

Run layout: `run.json`, `majorana/`, `nickelates/`, `findings.json`, `summary.md`. Each topic contains engine search artifacts at its root plus `commands.jsonl`, `coverage-stats.log`, `evidence.csv`, `report.md`, `provenance.json`; Majorana also has `predictions.json`. Optional citation graphs and resolved records stay inside their topic. Use existing schema-v2 provenance rules; keep engine receipts immutable and record the provenance validator's actual result, including failure, in commands.

`run.json` records campaign identity/environment/binary fingerprint, purpose, authorized ceilings, and per-topic status (`complete`, `partial`, `blocked`, `failed`) with reasons and actual usage/unknowns. `commands.jsonl` records argv, start/end, exit code and local stdout/stderr artifact paths (redacted). Each attempted command's output paths must exist inside the campaign root. A failed-before-search topic may omit engine-created files only when its status and command evidence explicitly explain that absence. Empty searches are a valid observed result, not automatically success or failure.

`findings.json` is an array (empty allowed). Each finding has stable ID, topic IDs, category (`engine_bug_candidate`, `agent_error`, `upstream_failure`, `usability_gap`, `scientific_gap`), severity, observed vs expected, command/artifact references, reproduction steps, reproducibility (`unconfirmed`, `offline_reproduced`), and proposed next action. An engine bug needs a reduced offline reproduction before calling it confirmed; do not manufacture findings or TODO fixes to fill a quota. The validator checks references/shape, not whether the expected behavior is scientifically right. Summary ranks verified failures separately from unconfirmed opportunities and states all gates and unsupported outputs.

For `--run`, check referenced files/paths, topic statuses, required matrix headers, five prediction slots with evidence or explicit gaps, well-formed source locators and matching finding references; reject `engine_verified` declarations on host synthesis. Do not compare engine receipt internals with hand-written approximations: the runbook invokes the real `rforge provenance validate` and records its result. A structurally valid partial report remains scientifically incomplete, clearly labeled in summary and final handoff.

Test seam: `unittest` temporary directories with synthetic case/run artifacts only, no scientific result fixtures. Tests: `test_case_packet` (missing case file red, then real packet), `test_partial_run` (missing/gated artifacts explicitly accounted for), `test_rejects_unsafe_or_overclaimed_run` (missing output, traversal/symlink escape, unsupported provenance, missing condition headers, dangling finding reference), `test_complete_run`. Tests create their own synthetic runs; no live calls, credentials or saved copyrighted text. Red → minimal green → refactor; include a synthetic-only disclaimer in test fixtures.

The runbook supplies exact bounded commands based on the current CLI/help, commands to write query files from the case packet, coverage/brief/provenance checks, output locations, resume limits, and safe human-gate stops. No general runner, installer or host integration. Existing CLI tests provide engine safety guards; packet validation is not a new engine qualification.

### V-FS2 — Two live dossiers and triage (Needs decision)

Consumes the packet, checker and runbook unchanged. It writes only `research/frontier-science/` and is the sole writer for that campaign root. Do not overwrite a prior campaign or change product code while collecting the baseline. If a baseline engine failure blocks later stages, preserve it and continue only independent safe work; do not quietly test an unrecorded patched binary.

This task is **not eligible now**. Before execution the owner/operator supplies a bounded live agent/token/time/spend allowance and authorizes the proposed API scope. Full-text acquisition, screening/claim acceptance, privacy review and package export remain separately gated even after live retrieval is authorized. Without those approvals use legal public metadata/abstract evidence only, label draft conclusions accordingly, and record full-text gaps; never route around the gates. No blocker is imposed on V-FS1.

After authorization, run Majorana then nickelates. Collect the named outputs, source evidence and command receipts, and validate the dossier structure offline. Exercise paper resolution/citation expansion only on retrieved identities within the separate ceilings. `brief` is a mechanical corpus summary, not the scientific verdict. Compare the two runs' operational failure modes, not their physical measurements. Preserve source-retrieval failures and negative/contradictory evidence as different categories.

A source-backed draft can legitimately conclude that the evidence cannot support a definitive answer. Source access limits must narrow claims. Five Majorana predictions are proposals for falsification, not asserted demonstrations. A partial or gated campaign is handed back honestly; the live task stays open if either topic was not actually attempted. Later improvements need separate scoped tasks, failing regression tests and existing full checks; no implementation is authorized by this plan.

## Verification and pre-mortem

**Verify:** packet/dossier checker plus explicit artifact-based acceptance in TODO. There is no invented scalar quality metric or measured baseline for this planning pass. The proposed checker and its tests do not exist yet and were not run.

**Guard for future code changes:** `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE make check`. Existing offline engine regression tests are listed in V-FS1. All Git subprocesses in new code must use `oss.IsolatedGitEnv()` or an equivalent scrubbed environment. Normal tests must never hit scholarly APIs.

Three likely failures and guards:

1. Missing evidence is silently promoted to scientific certainty: packet/partial-run tests require evidence columns, explicit gaps, draft status and no elevated host provenance; substantive support still requires source audit.
2. Interrupted/provider-failed searches reset the allowance or become fake saturation: reuse the existing budget/resume tests and record checkpoint caps/stop reasons/failures; do not infer coverage from result counts.
3. Results overwrite another campaign or references escape the artifact root: no-overwrite runbook instruction, temporary-directory checker tests for traversal/symlink escapes and dangling references, and a single writer per root.

Open authority: V-FS2 live budget/access approval only; not a reason to delay the offline packet. Scientific timeliness, available full text, replication and actual defects are execution findings, not assumptions to resolve by inventing literature during planning.

## Planning handoff

- Fresh read-only builder review: **nothing important left for V-FS1**; V-FS2 retains only the explicit live-authorization/dependency gates. No implementation tests were run by that reviewer.
- `goals.py plan-lint` reported zero errors. Planner's call: retain its one `directory-touches` warning for V-FS2 because campaign IDs and engine receipt filenames are generated; `research/frontier-science/` is a dedicated write root with a single writer, not a shared research-wide glob.
- Consumer check: V-FS1 is `ready_for_autowork` (exit 0); the whole two-task batch is `backlog_reconciled_not_eligible` (exit 2) until V-FS1 is complete and V-FS2's live authority is resolved. Readiness is not execution authorization or proof of live ownership.
- Planning validation: `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE make check` passed on the current tree. Future packet/checker tests remain unimplemented and unrun; this is not live research evidence.
- Narrow clerical repair: the existing V-B13-LIVE body was missing Goal and Sources despite its RF-BENCH ledger membership. Restore those fields only; do not change its scope, state, dependencies or live approval requirements.
- Next executable action: V-FS1, write and run `test_case_packet` against the absent case file before adding the offline packet. Stop after planning in this session; do not dispatch implementation or start live studies.
