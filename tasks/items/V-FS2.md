+++
schema = 1
id = "V-FS2"
title = "Run Majorana and nickelate stress dossiers and triage findings"
short = "Research two public case studies"
status = "open"
goal = "RF-BENCH"
section = "Now"
rank = 1000
depends_on = ["V-FS1"]
touches = ["research/frontier-science/"]
resources = []
kind = "implementation"
ownership = "worker"
scope = "After explicit live authorization and V-FS1/offline checks, execute one real Majorana/Comprehensive and one nickelates/Standard workflow on a recorded binary/agent snapshot. Main outputs: complete readable reports, precise source-backed citations, conditions/evidence tables, competing hypotheses, uncertainties and evidence-proportionate conclusions, plus per-topic citation-audit.json/reproduce.md. Keep bugs separately in findings.json. No synthetic/canned showcase, hidden adverse evidence, product fixes, automatic scientific approval, extra depth runs or 48-run benchmark changes."
acceptance = ["Given two genuine topic runs and their reports, the V-FS1 contract proven by `test_check.py::test_complete_run` validates real command/artifact references; both topics must actually run and have the report/source-audit/reproduction contract in the plan. A valid partial/failed-before-search fixture is never task completion or a public example.", "Given reports and machine-readable source audits, the contract proven by `test_check.py::test_rejects_unsafe_or_overclaimed_run` plus actual `rforge provenance validate` checks indexed output hashes/run linkage without elevating host judgments; the summary leads with research results and keeps traceable QC findings separate, including when none are found."]
checks = ["Run the V-FS1 checker tests and packet `check.py --cases docs/benchmark/frontier-science/cases.json --run <campaign-dir>` offline with Python 3; run actual `rforge provenance validate <topic-dir>/provenance.json` on each topic and retain real results; parse citation-audit.json as JSON and verify every declared report anchor/evidence-row/source reference; `git diff --check`. Record campaign-dir/binary path in run.json; do not assume PATH."]
exclusions = []
+++

- Sources: [public-examples handoff](docs/benchmark/frontier-science-plan.md#frontier-science-stress-test-plan); [benchmark boundary](docs/verifiable-engine-plan.md#verifiable-research-engine-plan).
- Owner decision: 2026-10-20: fix
- Budget: One shared V-FS2 allowance, not per topic/session/retry: USD 20, 1,000,000 input/output tokens (including reported cached input), 7,200 seconds from actual campaign execution start. Account for research/synthesis/helpers and preserve consumed usage across retries/resumes. Unknown usage is not zero. Use supported host accounting/deadline controls; if a metered operation cannot be safely bounded/accounted for, stop it and report the concrete gap on this task rather than waiving a limit or changing global runner settings.
- Owner decision: 2026-10-21: raise the shared V-FS2 token ceiling from 200,000 to 1,000,000 input/output tokens (cached input included), after attempt 1 used 403,202 tokens (USD 0.31 reported); the USD 20 and 7,200-second ceilings are unchanged, and usage already consumed still counts against the new ceiling.
- Consumes: `docs/benchmark/frontier-science/cases.json` (schema_version=1, purpose=exploratory_not_gold, ordered cases); `python3 docs/benchmark/frontier-science/check.py --cases PATH [--run DIR]` (read-only, exit 0 structurally complete / 1 invalid); `docs/benchmark/frontier-science/runbook.md` (bounded commands and human-gate stops).
- Produces: `research/frontier-science/<campaign-id>/` with `run.json` (campaign_id, started_at, complete real topic runs), both topic report.md/evidence.csv/provenance.json/citation-audit.json/reproduce.md, actual engine artifacts and separate findings.json; select the latest qualifying started_at, then campaign_id.
- Ownership: Owner/operator authorizes live budget and human gates; research agent is sole writer for the campaign, including source/report corrections before closeout.
- Live checks: Campaign API scope and model/token/time ceilings are explicitly approved above; check actual runtime access/accounting without inventing credentials or zero usage. Full-text acquisition, privacy review, accepted claims and package export remain separately gated. Inspect accessible primary records/passages, remove or narrow unsupported material claims, preserve contrary evidence and access gaps, and never hide uncertainty to make the product look better.
- Next action: Re-run the existing offline packet checks, record the approved shared ceilings and actual campaign start/accounting in authorization.md/run.json, then let the existing Autowork owner execute the same two-topic packet and report/audit contract. Stop at the first budget limit. Preserve unfinished/gated outcomes, consumed usage and checkpoint identity; do not fabricate research or reset budget to repeat the campaign.
- Resume authorization: 2026-10-21 owner: allow one more resume of the interrupted Majorana search beyond the one already permitted (both earlier runs hit the required 120-second timeout at 75/100 connector calls). The 120-second timeout, the accepted complete-topic contract and the shared V-FS2 budget (USD 20, 1,000,000 tokens, 7,200 seconds, consumed usage counted) are unchanged; if this resume also times out or leaves the topic incomplete, stop and report it as an owner question again.
