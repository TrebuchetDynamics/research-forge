# Reproduce / inspect nickelates

This document distinguishes mechanical inspection from host scientific synthesis and human approval. The saved report is an **agent_reported draft**, not CLI-generated scientific truth. [Report](report.md), [source audit](citation-audit.json), [evidence](evidence.csv), [primary metadata checks](primary-checks.json), [campaign authorization](../authorization.md), [run state](../run.json).

## Tested snapshot and caveat

Campaign `frontier-20261021-vfs2-live`; original start `2026-10-21T15:06:50.640852Z`. Source revision **04d4e87d0ed2f66d95f92aa741e39892fd6ab2e5**, original tracked diff SHA-256 **e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855**; dirty=true because runtime untracked metadata, no untracked Go build inputs recorded. Packet base restored onto **d58b0da456c9758ec3435736f66a1c232a5e4e5e**. See setup/build/recovery logs at campaign root; no new engine build during synthesis.

Original executable SHA-256 **52bcdd5d0fb84f90551113a8c2cd1f516e7d3671de25ce0c685cb3e342b90d7a** is no longer available. Rebuilt from that pinned source, current path **/var/tmp/aw-1000-d0648c07-1/vfs2-tools/rforge**, SHA-256 **8b869cb012fc7cc20b134976e9d8f599c5dce6e5d27b02a4bafc8b5196767308**. Its structured version is `{"version":"dev","commit":"unknown","date":"unknown"}`: source revision is external build evidence, not a falsely embedded version. Binary hashes differ; owner explicitly authorized disclosed continuation. This is an audit gap, not proof of byte equivalence. Historical original-engine receipts were not rewritten. A future build can differ again. The ephemeral executable is **not committed**.

Host environment: Linux recorded in run.json; Pi, **openai-codex/gpt-6.1-sol**. Engine retrieval/dedupe/resolution/expansion/statistics/provenance are genuine; selection, scientific reasoning, passage assessment and report writing are host work. Independent human scientific acceptance is absent.

## Offline inspection — no APIs, no spend

From repository root (Python standard library; compatible rforge executable needed only for actual engine validation):

```sh
export RUN="$PWD/research/frontier-science/frontier-20261021-vfs2-live"
export RF=/var/tmp/aw-1000-d0648c07-1/vfs2-tools/rforge
sha256sum "$RF"  # compare rebuilt hash above, not the unavailable original
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s docs/benchmark/frontier-science -p test_check.py -v
PYTHONDONTWRITEBYTECODE=1 python3 docs/benchmark/frontier-science/check.py --cases docs/benchmark/frontier-science/cases.json --run "$RUN"
PYTHONDONTWRITEBYTECODE=1 python3 "$RUN/validate-audits.py" "$RUN"
"$RF" --json provenance validate "$RUN/nickelates/provenance.json"
```

The packet checker checks structure/safe paths; the audit checker checks actual anchors, CSV row numbers and source identities/locators; real `rforge provenance validate` recomputes hashes/run-event linkage. None verifies scientific truth or grants a gate. Both topic indexes have actual hashes for reports/evidence/audit/reproduce/queries/manifest/checkpoint/statistics and other saved stable outputs, not themselves or mutable command/validator logs. Index level is **agent_reported**. Immutable historical receipts remain for history; do not validate their replaced historical output hashes as though current. The current applicable run IDs are in citation-audit.json and provenance.json. Runtime job records/locks were inventoried and moved outside the campaign only after checking that no immutable receipt outputs bind them.

## Actual stop, coverage and scope

Selected depth **standard**; selected queries in queries.txt and manifest; four explicit sources, limit **20**. Effective budget `{"depth": "standard", "max_query_variants": 10, "max_sources": 5, "max_api_calls": 50, "low_yield_threshold": 2, "saturation_queries": 3, "max_consecutive_errors": 5}`. Cumulative calls **24**, unique deduped records **254**, actual final stop **completed**, consecutive-low-yield **0**. Coverage is marginal yield, not relevance or recall. Provider failures remain in failures.jsonl, not interpreted as absent scientific phenomena.

Majorana history: initial and first resume each hit 120s process timeout, with 75 cumulative calls before recovery; owner authorized exactly one extra resume (two cumulative), which exited 0 at 100/100. Nickelates: no resume, terminal 24/50 with six provider failures. No histories or counts reset. Cap exhaustion is unrelated to withdrawn model budgets. All token/cost/campaign-time budgets are withdrawn; actual usage snapshots retain prior child plus parent and current child consumption, with unknowns not zero. Search/resolve/citation/process caps remain.

Primary abstract/page inspection used only retrieved source identities and the primary retraction notice. No PDFs/full texts or raw experimental data acquired. Source checks and short excerpts are host-attributed in primary-checks.json; explicit page opens and unknown HTTP attempts are separately recorded in run.json. Preprint/journal links are not a claim of equivalent versions or complete correction coverage.

## Opt-in live replay — do not run as a normal test

Requires a **new explicit authorization** for existing provider access and scope, compatible pinned-source build/toolchain and network, a **new output directory**, and a human/host to inspect failures and scientific support. Live providers may rate-limit, charge under existing access, change metadata, dedupe ordering or version content; results are **not byte deterministic**. No new account/purchase is authorized by these instructions. Do not overwrite this campaign or resume its exhausted/finished checkpoint. To construct a new run, use the exact selected queries and the search argv below but replace `--out` and `--queries` with the new campaign paths; apply a 120s process cap and preserve logs. A historical resume authorization is not permission for new retries. Keep resolution/expansion ceilings and retrieve seed identities before resolving/expanding. Historical fixed seed identities do not justify an unretrieved new seed.

A host must independently select relevant/adverse evidence, check primary passages/version notices, preserve measurement definitions and write a new source audit; CLI `brief` does not recreate these semantic judgments. Human-owned acquisition, accepted screening/claims/suggestions, privacy, scientific approval and export remain separate gates. No gated commands or approvals were run. Do not automatically replay source pages, search, acquire PDFs or export a package during offline inspection.

## Exact recorded operations and exit outcomes

These are the actual argv/times/exits for historical engine operations (absolute paths preserved as evidence, not portable guarantees). Each was bounded with `timeout --signal=TERM --kill-after=5s 120s`; mutable logs remain referenced by commands.jsonl. Final actual validator commands/results are appended there with distinct IDs. No command receipt is hand-edited.

```json
{
  "id": "capabilities",
  "stage": "preflight",
  "argv": [
    "/var/tmp/aw-1000-d0648c07-1/vfs2-tools/rforge",
    "capabilities",
    "--json",
    "--min-rforge-version",
    "0.1.22",
    "--require",
    "search=1",
    "--require",
    "paper=1",
    "--require",
    "brief=1",
    "--require",
    "compare=1",
    "--require",
    "provenance=1"
  ],
  "start": "2026-10-21T15:11:18Z",
  "end": "2026-10-21T15:11:18Z",
  "exit_code": 0,
  "stdout": "nickelates/capabilities.stdout.log",
  "stderr": "nickelates/capabilities.stderr.log"
}
```

```json
{
  "id": "search-1",
  "stage": "search",
  "argv": [
    "/var/tmp/aw-1000-d0648c07-1/vfs2-tools/rforge",
    "--json",
    "search",
    "batch",
    "--out",
    "/home/xel/git/research-forge/.worktrees/autowork-V-FS2/research/frontier-science/frontier-20261021-vfs2-live/nickelates",
    "--queries",
    "/home/xel/git/research-forge/.worktrees/autowork-V-FS2/research/frontier-science/frontier-20261021-vfs2-live/nickelates/queries.txt",
    "--sources",
    "openalex,crossref,semantic-scholar,arxiv",
    "--depth",
    "standard",
    "--limit",
    "20",
    "--max-requests",
    "50",
    "--continue-on-error",
    "--stats"
  ],
  "start": "2026-10-21T15:11:18Z",
  "end": "2026-10-21T15:12:24Z",
  "exit_code": 0,
  "stdout": "nickelates/search-1.stdout.log",
  "stderr": "nickelates/search-1.stderr.log"
}
```

```json
{
  "id": "coverage",
  "stage": "coverage",
  "argv": [
    "/var/tmp/aw-1000-d0648c07-1/vfs2-tools/rforge",
    "--json",
    "search",
    "stats",
    "--dir",
    "/home/xel/git/research-forge/.worktrees/autowork-V-FS2/research/frontier-science/frontier-20261021-vfs2-live/nickelates"
  ],
  "start": "2026-10-21T15:12:24Z",
  "end": "2026-10-21T15:12:24Z",
  "exit_code": 0,
  "stdout": "nickelates/coverage.stdout.log",
  "stderr": "nickelates/coverage.stderr.log"
}
```

```json
{
  "id": "brief",
  "stage": "brief",
  "argv": [
    "/var/tmp/aw-1000-d0648c07-1/vfs2-tools/rforge",
    "brief",
    "/home/xel/git/research-forge/.worktrees/autowork-V-FS2/research/frontier-science/frontier-20261021-vfs2-live/nickelates",
    "--out",
    "/home/xel/git/research-forge/.worktrees/autowork-V-FS2/research/frontier-science/frontier-20261021-vfs2-live/nickelates/brief.md"
  ],
  "start": "2026-10-21T15:12:24Z",
  "end": "2026-10-21T15:12:24Z",
  "exit_code": 0,
  "stdout": "nickelates/brief.stdout.log",
  "stderr": "nickelates/brief.stderr.log"
}
```

```json
{
  "id": "paper-1",
  "stage": "paper",
  "argv": [
    "/var/tmp/aw-1000-d0648c07-1/vfs2-tools/rforge",
    "--json",
    "paper",
    "10.1038/s41586-023-06408-7",
    "--out",
    "/home/xel/git/research-forge/.worktrees/autowork-V-FS2/research/frontier-science/frontier-20261021-vfs2-live/nickelates/paper-1.json"
  ],
  "start": "2026-10-21T15:13:07Z",
  "end": "2026-10-21T15:13:08Z",
  "exit_code": 0,
  "stdout": "nickelates/paper-1.stdout.log",
  "stderr": "nickelates/paper-1.stderr.log"
}
```

```json
{
  "id": "paper-2",
  "stage": "paper",
  "argv": [
    "/var/tmp/aw-1000-d0648c07-1/vfs2-tools/rforge",
    "--json",
    "paper",
    "10.1038/s41563-021-01142-9",
    "--out",
    "/home/xel/git/research-forge/.worktrees/autowork-V-FS2/research/frontier-science/frontier-20261021-vfs2-live/nickelates/paper-2.json"
  ],
  "start": "2026-10-21T15:13:08Z",
  "end": "2026-10-21T15:13:08Z",
  "exit_code": 0,
  "stdout": "nickelates/paper-2.stdout.log",
  "stderr": "nickelates/paper-2.stderr.log"
}
```

```json
{
  "id": "citations-1",
  "stage": "citations",
  "argv": [
    "/var/tmp/aw-1000-d0648c07-1/vfs2-tools/rforge",
    "--json",
    "citations",
    "expand",
    "--source",
    "openalex",
    "--paper",
    "W4384028978",
    "--direction",
    "both",
    "--depth",
    "1",
    "--max-records",
    "30",
    "--max-api-calls",
    "6",
    "--retry-budget",
    "0",
    "--out",
    "/home/xel/git/research-forge/.worktrees/autowork-V-FS2/research/frontier-science/frontier-20261021-vfs2-live/nickelates/graph-1.json"
  ],
  "start": "2026-10-21T15:13:08Z",
  "end": "2026-10-21T15:13:09Z",
  "exit_code": 0,
  "stdout": "nickelates/citations-1.stdout.log",
  "stderr": "nickelates/citations-1.stderr.log"
}
```

```json
{
  "id": "coverage-final",
  "stage": "coverage",
  "argv": [
    "/var/tmp/aw-1000-d0648c07-1/vfs2-tools/rforge",
    "--json",
    "search",
    "stats",
    "--dir",
    "/home/xel/git/research-forge/.worktrees/autowork-V-FS2/research/frontier-science/frontier-20261021-vfs2-live/nickelates"
  ],
  "start": "2026-10-21T16:43:07.167978+00:00",
  "end": "2026-10-21T16:43:07.191931+00:00",
  "exit_code": 0,
  "stdout": "nickelates/coverage-final.stdout.log",
  "stderr": "nickelates/coverage-final.stderr.log"
}
```

```json
{
  "id": "brief-final",
  "stage": "brief",
  "argv": [
    "/var/tmp/aw-1000-d0648c07-1/vfs2-tools/rforge",
    "brief",
    "/home/xel/git/research-forge/.worktrees/autowork-V-FS2/research/frontier-science/frontier-20261021-vfs2-live/nickelates",
    "--out",
    "/home/xel/git/research-forge/.worktrees/autowork-V-FS2/research/frontier-science/frontier-20261021-vfs2-live/nickelates/brief-final.md"
  ],
  "start": "2026-10-21T16:43:07.230452+00:00",
  "end": "2026-10-21T16:43:07.289365+00:00",
  "exit_code": 0,
  "stdout": "nickelates/brief-final.stdout.log",
  "stderr": "nickelates/brief-final.stderr.log"
}
```
