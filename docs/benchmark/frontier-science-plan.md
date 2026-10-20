# Frontier-science stress-test plan

Status: Accepted

This is the accepted handoff for **public, high-quality scientific research examples**, with scientific stress-testing as supporting quality control. The historical filename/heading is retained so existing task/source links remain valid. The owner's confirmed priorities below supersede earlier bug-hunting-first descriptions and the old prohibition on repository publication. Remaining operating defaults are **planner's calls**, not additional owner decisions. Planning itself does not run research or authorize live spend.

Tasks: [V-FS1](../../TODO.md#now), [V-FS2](../../TODO.md#needs-decision) and [V-FS3](../../TODO.md#next). This slice supports RF-BENCH without changing its held-out-ground-truth definition, existing benchmark task IDs or [release gate](../verifiable-engine-plan.md#acceptance).

## Confirmed owner direction — public examples first

Status: Accepted

Owner instruction: **“Public research examples are the primary deliverable. Scientific stress-testing is supporting quality control.”**

1. Publish two exemplary scientific case studies in this repository: Majorana zero modes at **Comprehensive** depth and nickelate superconductivity at **Standard** depth, one genuine run per topic. Do not introduce depth-comparison runs or modify the existing 48-run benchmark.
2. Each must be a complete, readable, scientifically rigorous report, with verified sources, precise citations, evidence tables, competing hypotheses, limitations, uncertainty and conclusions supported by the available evidence.
3. Preserve commands, configuration, provenance and machine-readable artifacts so a scientist or developer can read the findings, inspect their support and reproduce the workflow. Make both examples discoverable from the repository README.
4. Outputs must come from actual ResearchForge execution, not manually fabricated demonstrations or synthetic literature. Clearly attribute host-agent synthesis and mechanical engine operations separately.
5. Keep bugs and tool limitations separate from the scientific reports. Confirmed software defects become regression-test/fix tasks in a later phase; no defect quota or fixes during the baseline campaign.
6. Complete V-FS1 and offline validation first. Live execution remains gated on explicit API scope and agent/token/time/spend authorization; no automatic scientific approvals or bypass of existing acquisition/privacy/claim/package gates.
7. Optimize scientific quality, readability, citation integrity and reproducibility—not maximum tool usage, report length or a favorable product narrative. Never conceal uncertainty, unfavorable evidence or failures. A well-supported conclusion that the available evidence is insufficient is a successful showcase.

**Acceptance:** a visitor can find both genuine case studies from README, read supported findings and rival explanations, inspect cited evidence and real rforge receipts, and follow honest offline-inspection and opt-in live-reproduction instructions. A structurally valid but failed-before-search or synthetic dossier is never promoted as a finished example.

**Current worker boundary:** V-FS1 was implemented and under Autowork verification when this clarification was recorded. Preserve its four-file contract, tests, synthetic-only fixtures and active worktree. The new public curation step belongs to V-FS3, not a competing rewrite of V-FS1. V-FS2 remains gated.

## Brief and boundaries

**Size:** small project: offline preparation (V-FS1), a separately gated live campaign producing genuine reports (V-FS2), then offline source-audit curation and repository discovery (V-FS3).

**Goal:** demonstrate what ResearchForge can produce through two public, readable, scientifically rigorous case studies. Exercise search → dedupe → paper resolution/citation expansion → evidence-led report → provenance validation as the means; discovering bugs is secondary quality control.

**Success:** two source-backed reports with evidence tables, competing hypotheses, precise citations, uncertainties and reproducible artifacts, discoverable from README. Source evidence may support an explicitly inconclusive answer. Bug findings live separately; neither defect count nor an exciting scientific verdict is required. Failed/gated stages stay visible and are not advertised as finished examples.

**Non-goals:** implement fixes during the campaign; build a general benchmark framework; execute/modify the 48-run Pi pilot; compare depth levels through additional runs; add connectors, models or services; run a pooled meta-analysis; acquire PDFs, import screening approvals, accept suggestions/claims or export packages without existing human gates; push, release or deploy from this planning handoff.

**Anti-goals:** canned or synthetic showcase evidence; manually invented measurements/citations/receipts; hiding adverse findings, uncertainty or tool failures for marketing; unsupported exciting conclusions; experimental signatures presented as conclusive proof; pressure/material/measurement conditions collapsed into one number; provider outages presented as negative scientific evidence; a mechanical provenance pass presented as scientific approval; frontier runs promoted to gold sets, recall estimates or release evidence.

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

1. **Chosen: two genuine public case studies over existing CLI/services.** Directly demonstrates useful research, with transparent supporting QC and no invented gold inclusion set. Preserve the exploratory packet's machine contract; add reader-facing discovery after actual research, not more test equipment in place of reports.
2. **Run the existing paired 48-run pilot now.** Rejected for this slice: different purpose, gold-set prerequisites and live qualification/spend gates. Leave its accepted design intact.
3. **Build a new scored benchmark across all twelve topics.** Deferred: increases cost and framework work before the two-topic trial establishes useful failure modes; no defensible gold inclusion sets are provided.

No new ADR is needed: these choices are reversible and do not change production architecture.

## Planner's calls

- **Initial pair/order:** Majorana zero modes, then nickelate superconductivity. The first targets signature false positives; the second targets quantitative comparability. Emergent spacetime, Hubble tension, black-hole information and Gaussian primes remain proposed later topics, not queued work.
- **Unit of work:** one host-agent-assisted CLI research run per topic using the same recorded binary and agent/model identity. No without-rforge condition, repetitions, statistical effect estimates or claim of tool superiority.
- **Depth (now confirmed by owner):** Majorana Comprehensive; nickelates Standard. One initial run per topic, no Standard/Comprehensive matched comparison or automatic extra run. Query plans and caps are ceilings, not targets; stop when evidence and the engine's actual stop signal warrant it.
- **Search defaults:** four `scholarly-fast` sources, limit 20 per query/source; Majorana at most 100 connector search calls total; nickelates at most 50. No automatic increase. A needed nickelate Comprehensive escalation is recorded as an unfinished follow-up, requiring a new allowance before more calls. These are proposed live ceilings, not permission to spend.
- **Additional live bounds:** at most 3 retrieved citation seeds per topic, depth 1, `--max-records 30 --max-api-calls 6 --retry-budget 0` per expansion; at most 5 `paper` resolutions per topic. Do not count these as part of `search batch`'s request cap. All network commands get a 120-second process timeout, preserving partial output/checkpoints; at most one resume per search within the original total allowance, never an unbounded retry loop. Report HTTP attempts only when observable; otherwise mark unknown.
- **Credentials/cost:** existing authorized provider access only; no paid account setup, new subscriptions, configuration changes or purchases. V-FS2 stays in Needs decision until an operator authorizes the live scope, agent/token/time/spend budget and any needed human review. Runtime auth/quotas are checked then, not fabricated in planning.
- **Artifacts:** prepare `docs/benchmark/frontier-science/`; live results go only under `research/frontier-science/<campaign-id>/`. Refuse to overwrite an existing campaign root. Record UTC start, binary SHA-256/version, Git revision plus dirty status/diff fingerprint, OS, agent/model, budgets and commands. Never copy `.env`, credentials, full session secrets or unrestricted copyrighted text.
- **Scientific status and genuine generation:** rforge must actually perform the recorded searches, dedupe, identity/citation work and provenance checks; retain those real outputs. The engine does not independently author scientific judgments: host-agent narrative synthesis over retrieved evidence is allowed and must remain `agent_reported`, never falsely branded engine-verified or peer-reviewed. The ban on manually authored demonstrations means no canned research, fabricated data or synthetic receipts as a showcase—not concealing the host's legitimate synthesis role. Use `unknown`/null with reasons for unavailable source text, uncertainty or replication; inspect accessible correction/retraction/version records and disclose gaps. No self-approved scientific conclusions.
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

Consumes the V-FS1 packet, checker and runbook, preserving their existing exploratory machine contract. It writes only `research/frontier-science/` and is sole writer for its fresh campaign root. The confirmed public-example requirements in this plan supersede bug-hunting-first language and the old runbook sentence prohibiting repository publication; they do not supersede any live/approval gate or engine receipt. Do not restart V-FS1, overwrite a prior campaign or change product code while collecting the baseline. If an engine failure blocks a stage, retain the failure, narrow unsupported claims and continue only safe independent work; never silently switch to an unrecorded patched binary.

This task is **not eligible now**. Before execution the owner/operator supplies a bounded live agent/token/time/spend allowance and authorizes the proposed API scope. Full-text acquisition, screening/claim acceptance, privacy review and package export remain separately gated even after live retrieval is authorized. Without those approvals use legal public metadata/abstract evidence only, label draft conclusions accordingly, and record full-text gaps; never route around the gates. No blocker is imposed on V-FS1.

After authorization, run Majorana then nickelates. Produce the scientific reports described below as the principal output, with actual source records/evidence and command receipts; validate the dossier structure offline. Exercise paper resolution/citation expansion only on retrieved identities within the separate ceilings. `brief` is a mechanical corpus summary, not the research report. Compare operational failure modes only in separate QC findings; do not compare physical measurements across topics or claim a depth effect. Preserve source-retrieval failures and negative/contradictory scientific evidence as different categories.

A report can legitimately conclude that evidence is insufficient. Source access limits must narrow claims. Five Majorana predictions remain proposed falsification tests, not demonstrations; unsupported slots remain honest gaps. A failed-before-search/header-only dossier is retained as partial failure evidence, not a substitute for either genuine research example. V-FS2 is not complete merely because partial structure validates: both real topic workflows must have run, the source audit/reproduction files below must exist and the two readable reports must address the questions from actual evidence. If access/budget/approval prevents that, retain unfinished status and specify the unmet outcome—never fabricate content or make extra unauthorized runs.

### V-FS2 report, source audit and reproduction contract

Each `<campaign-id>/<topic>/report.md` is a complete research case study, not a test transcript or checklist. Use: research question and concise findings; scope/search method and cutoff; evidence table; competing hypotheses and contradictory findings; interpretation separating observations/theory/replication; limitations and uncertainty; evidence-proportionate conclusion; numbered references and reproduction links. Keep source citations adjacent to material claims. Length follows the evidence, not a quota. Clearly label host synthesis, source-access limits and any non-peer-reviewed status. Include unfavorable and inconclusive findings with the same standards as favorable ones.

Add `citation-audit.json` and `reproduce.md` to each topic (normal extra outputs; do not change schema-v2 or immutable run receipts). Index these new files with actual hashes in the topic provenance index, then run the real engine validator. Do not index mutable command/validator logs or the provenance index itself. A failed engine validation remains a QC finding and prevents promotion as an exemplary finished study until addressed through an authorized later fix, not hidden by a structural pass.

`citation-audit.json` is an object with `topic`, `assessed_by: {kind: "host_agent", identity: <recorded agent/model>}`, `engine_run_ids` (actual topic run IDs), and `claims` array. Each material scientific/numerical claim in the report has a stable `id`, an actual `report_anchor`, `claim`, `evidence_rows` (1-based data rows, excluding CSV header), `source_id`, `source_locator`, `source_location`, `source_status` (published/preprint/corrected/retracted/unknown), `support` (supported/disputed/insufficient), `rationale`, and `access_limitations` string array. Verify title/authors/year/identifier and quotation/location against accessible primary records or source text; do not infer a measurement from a title. Distinguish reference identity, passage support, host semantic assessment and human approval. An inaccessible check is an explicit limitation, not a passing check. A claim of insufficient evidence can be fully supported by a documented gap; remove or narrow unsupported assertions rather than relabeling their audit status. References and evidence records remain inspectable; never redistribute uncleared full text.

`reproduce.md` records the tested engine/build revision and configuration, prerequisites, exact query/source/depth/cap arguments and actual stop reasons. Separate **offline inspection** (saved files, hashes, real provenance/checker commands, no API calls) from **opt-in live rerun** (may cost money, requires authorization and can return changed literature). Say which steps require a host agent or human; do not imply CLI commands independently recreate semantic judgments or that live providers are byte-deterministic. Do not execute an extra run just to write these instructions.

Campaign `summary.md` leads with links to the two research reports and a concise scientific overview; retain the V-FS1-required QC headings as a secondary section. Keep defect reports in `findings.json`, not mixed into scientific verdicts. Preserve command failures, limits and true uncertainty. No failure quota. Binaries, runtime jobs/locks, secrets and unauthorized full text stay outside committed public artifacts.

Produces for V-FS3: a genuine campaign at `research/frontier-science/<campaign-id>/` with `run.json` (`campaign_id`, `started_at`, topic state and actual engine identity), two topic `report.md`, `evidence.csv`, `provenance.json`, `citation-audit.json`, `reproduce.md`, real engine artifacts/receipts and separate `findings.json`. No new protocol framework or duplicate research directory is needed.

### V-FS3 — Curate and expose the genuine public examples

Runs offline only after V-FS1 and V-FS2 finish. Owns `README.md`, `research/frontier-science/README.md`, `research/frontier-science/examples.json`, and `internal/docs/frontier_examples_test.go` (`Resources: agent-docs`). It links the original reports/artifacts; do not copy or rewrite scientific outputs/engine receipts, launch new research, silently repair the engine, or advertise empty/failed/synthetic runs as completed examples. V-FS2 owns source/report corrections before its closeout. If curation discovers a substantive evidence problem, stop promotion of the affected study and return the exact issue to its existing task rather than inventing a passing audit.

Find the saved campaign without coordinator input: inspect immediate subdirectories under `research/frontier-science/` with `run.json`; select the latest by parsed UTC `started_at`, breaking ties lexically by `campaign_id`, among actual complete two-topic campaigns with the required files and passing real provenance checks. Never use test fixtures or `examples/open-data-project/` as research evidence. If none qualifies, report the unmet V-FS2 prerequisite; do not synthesize a placeholder campaign. Completion is recorded only when both reports qualify; a valid evidence-insufficient scientific conclusion is not a failed workflow.

The root README gains a short "Scientific research examples" section linking directly to both reports and the examples index, with one-sentence questions/depth labels and no invented headline results. The index explains report/artifact locations, actual rforge-vs-host roles, limitations, how to inspect offline and how to request a live rerun. Link separate QC findings without burying unfavorable scientific evidence. No marketing rewrite of unrelated README sections.

`examples.json` contains `schema_version: 1` and exactly two ordered `examples` (`id`: majorana then nickelates; `depth`: comprehensive then standard). Each has repository-relative `campaign_dir`, `report`, `evidence`, `provenance`, `citation_audit`, `reproduce` paths into the same real campaign. This is a small discovery manifest, not a new research store or automatic scientific approval.

Test seam: `internal/docs/frontier_examples_test.go::TestFrontierExamplesDiscoverable` first fails on the missing index/README section, then checks both links, depth labels and safe existing targets; `TestFrontierExamplesRejectIncompleteOrSynthetic` uses synthetic temporary trees to reject missing/failed engine runs, wrong-topic/missing audit or evidence references, empty evidence, unresolved audit anchors, escaped/symlink paths and changed output hashes. A valid synthetic test fixture is only validator test input, never a showcase. The real-artifact branch resolves the manifest's recorded engine IDs/events and actual provenance outputs, checks report/evidence/audit reference consistency, and rejects a mere header-only dossier. Tests stay offline and never launch a live research command. Test helper structure and parsing are reversible implementer choices; metadata checks cannot establish scientific truth. Source/semantic review remains a separately attributed assessment, not the test's scientific approval.

Before promotion, inspect every material claim and cited location using the saved audit and accessible authorized source evidence, distinguish competing explanations, preserve adverse findings and access gaps, and label the limits of host semantic assessment. Use only rights-cleared metadata/short excerpts; obey any applicable privacy/final-claim/export gate and do not claim a peer review or human approval that did not occur. This task makes reports discoverable in the repository; no push, release, package export or new spend is implied. Confirmed software defects remain proposed later red-green-refactor work, not part of curation.

## Verification and pre-mortem

**Verify:** packet/dossier checker plus explicit artifact-based acceptance in TODO. There is no invented scalar quality metric or measured baseline for this planning pass. The proposed checker and its tests do not exist yet and were not run.

**Guard for future code changes:** `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE make check`. Existing offline engine regression tests are listed in V-FS1. All Git subprocesses in new code must use `oss.IsolatedGitEnv()` or an equivalent scrubbed environment. Normal tests must never hit scholarly APIs.

Three likely failures and guards:

1. Missing evidence is silently promoted to scientific certainty: packet/partial-run tests require evidence columns, explicit gaps, draft status and no elevated host provenance; substantive support still requires source audit.
2. Interrupted/provider-failed searches reset the allowance or become fake saturation: reuse the existing budget/resume tests and record checkpoint caps/stop reasons/failures; do not infer coverage from result counts.
3. Results overwrite another campaign or references escape the artifact root: no-overwrite runbook instruction, temporary-directory checker tests for traversal/symlink escapes and dangling references, and a single writer per root.

Open authority: V-FS2 live budget/access approval only; not a reason to delay the offline packet. Scientific timeliness, available full text, replication and actual defects are execution findings, not assumptions to resolve by inventing literature during planning.

## Planning handoff

The prior two-task builder/lint/readiness receipt concerned the original bug-hunting-first scope; it is not evidence for the newly confirmed publication requirements. Preserve completed work and live ownership, and validate the affected V-FS2/V-FS3 handoff before reporting it ready. V-FS1 is already under Autowork verification; do not rerun or mark it complete from planning. Follow V-FS1 with explicit V-FS2 live authorization, then offline V-FS3 curation. No new live spend or extra depth-comparison run is authorized.

Planner's call: retain the V-FS2 `directory-touches` warning because campaign IDs/receipt filenames are generated and its dedicated root has one writer; V-FS3 shares that tree only through a dependency on V-FS2. The four-file V-FS3 write set is exact and its README edit declares `Resources: agent-docs`. Required full checks remain `make check`; new curation tests are planned, not executed evidence. The existing V-B13-LIVE clerical metadata repair and all unrelated task history remain unchanged.

The fresh read-only builder check on the revised V-FS2/V-FS3 bodies found **nothing important left** beyond explicit live authorization and task dependencies. It found no accidental blocker or source-ownership conflict and did not rerun/alter V-FS1. Delivery of this handoff means a scoped local planning commit for the existing runner to consume, not a request to start another worker, authorize live research, push or publish unfinished reports.
