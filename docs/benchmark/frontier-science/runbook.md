# Frontier-science exploratory packet

This is **exploratory, not gold**: no recall denominator, release qualification,
causal tool-superiority claim, or scientific approval. Follow the
[accepted plan](../frontier-science-plan.md) and the **repository**
[research skill](../../../skills/research-forge/SKILL.md), not an installed copy.
V-FS1 is offline preparation only. **Do not run the live sections until V-FS2's
operator authorizes API scope and agent/model, token, elapsed-time and spend
allowances.** These ceilings are not permission to spend. Use only existing
approved access; no account purchases, host configuration changes or new services.

## Offline packet check

From the repository root, Python standard library only:

```sh
python3 -m unittest discover -s docs/benchmark/frontier-science -p test_check.py -v
python3 docs/benchmark/frontier-science/check.py --cases docs/benchmark/frontier-science/cases.json
```

The checker is read-only, makes no subprocess/network calls and exits 0 for a
structurally valid packet/dossier (including honestly partial dossiers), 1 for
invalid artifacts. It does not verify scientific truth or emulate engine receipt
validation. Synthetic fixtures are not literature or genuine engine receipts.
The case cards use required machine-readable `evidence_requirements` identifiers
for the scientific boundaries; `evidence_notes` explains them for the host. The
checker requires those identifiers, not a placeholder or a judgment about prose.

## Authorized live setup — one new campaign only

Use Bash from the repository root. Pick a new safe campaign ID; never overwrite
an existing root, re-use another campaign, or change product code during a run.
Run Majorana first, then nickelates, with the same binary and host agent/model.
Record the operator's authorization and numeric token/time/spend allowances in
`authorization.md` under the campaign root before any API calls. Stop at those
allowances even if connector caps remain. Do not copy credentials, `.env`, full
sessions, unrestricted full text or secret-bearing command arguments into logs.

```sh
export CAMPAIGN_ID=frontier-YYYYMMDD-unique
[[ "$CAMPAIGN_ID" =~ ^[A-Za-z0-9_-]+$ ]] || exit 1
export RUN="$(pwd)/research/frontier-science/$CAMPAIGN_ID"
mkdir -p research/frontier-science
mkdir "$RUN" || exit 1                 # exclusive; never mkdir -p "$RUN"
mkdir "$RUN/majorana" "$RUN/nickelates" "$RUN/bin"
export CASES="$(pwd)/docs/benchmark/frontier-science/cases.json"
export RF="$RUN/bin/rforge"
env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go build -o "$RF" ./cmd/rforge
sha256sum "$RF"                        # copy exact hash to run.json
"$RF" --json version > "$RUN/version.json"
# Keep this snapshot fixed; fingerprint, do not publish the source diff.
env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE git rev-parse HEAD
env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE git status --porcelain
# Fingerprint tracked diff; separately inventory/hash any untracked build inputs.
env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE git diff --binary HEAD | sha256sum
uname -a
python3 - <<'PY'
import json, os
from pathlib import Path
packet = json.loads(Path(os.environ['CASES']).read_text())
for case in packet['cases']:
    path = Path(os.environ['RUN']) / case['id'] / 'queries.txt'
    with path.open('x') as stream:
        stream.write('\n'.join(case['queries']) + '\n')
PY
```

Retain build/version command outputs and exit codes as local setup evidence;
record UTC campaign start, OS, agent and exact model identity in `run.json`.
The `binary.version` object is `version.json`'s **data**, not its envelope or the
human version line. For an unavailable build/version, record the actual failure
and stop rather than silently using an installed release. Record Git revision,
dirty boolean and diff SHA-256; include untracked build-input names/hashes and
an explicit note if the tracked diff fingerprint alone cannot describe the build.

### Command receipts (no automated research runner)

For each command below, preserve actual argv, UTC start/end, exit code, and
redacted stdout/stderr. This small Bash receipt helper only captures one command;
it does not choose queries, retry, approve gates, or interpret results. Set `TOPIC`
manually in the required order. Do not use `set -e`: a nonzero command must remain
visible and the independent safe stages may still be attempted. Do not use `eval`.
All network commands have a 120-second timeout; the helper also bounds local
commands. Timeout exit 124/137 is not saturation. Keep partial checkpoints.

```sh
record() {
  local id="$1" stage="$2" start end code; shift 2
  [[ "$id" =~ ^[A-Za-z0-9_-]+$ ]] || return 1
  [[ "$TOPIC" == majorana || "$TOPIC" == nickelates ]] || return 1
  [[ ! -e "$RUN/$TOPIC/$id.stdout.log" && ! -e "$RUN/$TOPIC/$id.stderr.log" ]] || return 1
  start=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  if timeout --signal=TERM --kill-after=5s 120s "$@" \
      > "$RUN/$TOPIC/$id.stdout.log" 2> "$RUN/$TOPIC/$id.stderr.log"; then
    code=0
  else
    code=$?
  fi
  end=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  python3 - "$RUN" "$TOPIC" "$id" "$stage" "$start" "$end" "$code" "$@" <<'PY'
import json, sys
from pathlib import Path
root, topic, identity, stage, start, end, code, *argv = sys.argv[1:]
entry = dict(id=identity, stage=stage, argv=argv, start=start, end=end,
             exit_code=int(code), stdout=f'{topic}/{identity}.stdout.log',
             stderr=f'{topic}/{identity}.stderr.log')
with (Path(root) / topic / 'commands.jsonl').open('a') as stream:
    stream.write(json.dumps(entry) + '\n')
PY
  printf '%s exit=%s\n' "$id" "$code"
}
```

Use only non-secret argv; inspect/redact output before handoff. Record redactions
explicitly without editing engine receipts. Command output paths are relative to
**campaign root**, including the topic prefix; provenance output paths are
relative to **topic root**. Every referenced file must exist. Do not put stdout
outside the campaign, use `../`, or symlink to files outside it.

## Search → dedupe → resolve/expand → mechanical summary

First:

```sh
export TOPIC=majorana
record capabilities preflight "$RF" capabilities --json --min-rforge-version 0.1.22 \
  --require search=1 --require paper=1 --require brief=1 --require compare=1 --require provenance=1
```

If capability preflight fails, stop engine work for that topic and record the
failure; do not execute search. Otherwise:

```sh
record search-1 search "$RF" --json search batch --out "$RUN/$TOPIC" \
  --queries "$RUN/$TOPIC/queries.txt" --sources openalex,crossref,semantic-scholar,arxiv \
  --depth comprehensive --limit 20 --max-requests 100 --continue-on-error --stats
```

Inspect `manifest.json`, `batch-state.json`, `failures.jsonl` when present, and
search stdout. Dedupe is built into batch (DOI/title); do not invent a second
search/dedupe implementation. Record actual submitted queries and source skips,
failures, effective query/source/request caps, `requests_used`, coverage signal,
`stop_reason` and remaining gaps. Empty results are valid observations, not proof
of success, failure, absence of physics, or exhaustive coverage.

- `completed`: planned sweep attempted; inspect gaps and failures.
- `coverage_saturated`: stop; do not fill the query quota.
- `budget_exhausted`: stop at the cap.
- `interrupted` / `error_threshold`: record interruption/errors separately from
  scientific evidence. At most **one** resume, only after inspecting the cause,
  within the **original total** allowance. Never delete/reset the checkpoint.

For Majorana only, the one permitted resume command is:

```sh
record search-resume search "$RF" --json search batch --out "$RUN/$TOPIC" \
  --queries "$RUN/$TOPIC/queries.txt" --sources openalex,crossref,semantic-scholar,arxiv \
  --depth comprehensive --limit 20 --max-requests 100 --continue-on-error --stats --resume
```

After Majorana, set `TOPIC=nickelates`, run the same capability preflight, then:

```sh
export TOPIC=nickelates
record capabilities preflight "$RF" capabilities --json --min-rforge-version 0.1.22 \
  --require search=1 --require paper=1 --require brief=1 --require compare=1 --require provenance=1
```

Again, stop if preflight fails. Otherwise:

```sh
record search-1 search "$RF" --json search batch --out "$RUN/$TOPIC" \
  --queries "$RUN/$TOPIC/queries.txt" --sources openalex,crossref,semantic-scholar,arxiv \
  --depth standard --limit 20 --max-requests 50 --continue-on-error --stats
# Only if interrupted/error and still allowed: one resume, same total 50.
# record search-resume search "$RF" --json search batch --out "$RUN/$TOPIC" \
#   --queries "$RUN/$TOPIC/queries.txt" --sources openalex,crossref,semantic-scholar,arxiv \
#   --depth standard --limit 20 --max-requests 50 --continue-on-error --stats --resume
```

Nickelate Comprehensive escalation is an **unfinished follow-up**, not an
allowed automatic action. A new allowance is required even if the repository
skill would normally suggest escalation. Search caps count connector Search
invocations, not HTTP attempts. Client retries may cause extra HTTP requests:
record observed attempts, otherwise null plus reason. Separate counts for paper
resolution and citation expansion are mandatory; neither spends search units.

For **each** topic, while `TOPIC` still names that topic:

```sh
record coverage coverage "$RF" --json search stats --dir "$RUN/$TOPIC"
cp "$RUN/$TOPIC/coverage.stdout.log" "$RUN/$TOPIC/coverage-stats.log"
record brief brief "$RF" brief "$RUN/$TOPIC" --out "$RUN/$TOPIC/brief.md"
```

If stats cannot run, keep its failed command and write `unknown: <actual reason>`
to `coverage-stats.log`, never a fabricated zero. `brief.md` is a mechanical
corpus summary, **not** the scientific verdict in `report.md`.

Select only identities actually retrieved. At most **5 paper resolutions per
topic**, one identity per invocation; at most **3 citation seeds per topic**,
each depth 1, 30 records, 6 API calls, zero retry budget. No automatic graph
resume/retry. Choose the appropriate supported provider/identifier and record it:

```sh
# Set from a retrieved record, never a fabricated seed:
# export PAPER_ID=... SEED_ID=...
# record paper-1 paper "$RF" --json paper "$PAPER_ID" --out "$RUN/$TOPIC/paper-1.json"
# record citations-1 citations "$RF" --json citations expand --source openalex \
#   --paper "$SEED_ID" --direction both --depth 1 --max-records 30 \
#   --max-api-calls 6 --retry-budget 0 --out "$RUN/$TOPIC/graph-1.json"
```

Keep optional records/graphs and their receipts inside the topic directory.
Repeat manually with unique IDs only within the independent caps. Record skipped
or unavailable stages as gaps, not successful attempts. Stop on provider limits;
no unbounded loops, new credentials or purchased access.

## Host synthesis and human stops

Write `evidence.csv` using **all** `evidence_columns` from the corresponding case
card. Header-only is permitted for an empty search or a gated/failed topic; explain
why in report, status and summary. Otherwise each row has all cells: use
`unknown: <reason>` rather than blanks, inferred conditions, or fabricated errors.
A source locator is an HTTP(S) URL with host (no credentials/whitespace),
`doi:10.<registrant>/<suffix>`, `arxiv:<id>[vN]`, or
`artifact:<campaign-relative-file>#<location>`. Always keep quote/location too;
locator syntax is not support verification. Source-less gaps belong in the report,
not invented evidence rows. Preserve original units alongside explicit conversions
(extra CSV columns are allowed). Nickelate transition definitions, pressure
calibration, sample/dataset/group identity and measurement conditions must not be
collapsed. Do not pool incompatible transition temperatures or choose a mechanism
from them. For Majorana separate trivial alternatives and independent-group
replication from papers on the same device/data. Check accessible primary records
for corrections, retractions and version links; record inaccessible checks as gaps.

`report.md` is an **agent_reported draft**, containing signature comparison,
contradictions, uncertainty and a source-backed verdict separating demonstrated
observations, model-dependent interpretation and open questions. Missing full
text/uncertainty/replication narrows conclusions. Never self-approve conclusions.

`majorana/predictions.json` is an array of exactly five ordered slots, 1–5. A
supported proposal has `slot`, `status: "proposed"`, nonempty `prediction`,
`refuting_observation`, `source_rationale` and `source_locators` array. These are
falsifiable **proposals**, not demonstrations. An unsupported slot instead has
`slot`, `status: "unsupported"`, `reason`; keep the slot, never invent a prediction.

Full-text acquisition/PDFs, screening acceptance, suggestion/claim acceptance,
privacy review and package export **stop at existing human gates**, even after
live metadata retrieval approval. Do not run `oa fetch`, approve/import decisions,
accept suggestions or export packages on your own authority. Use legal public
metadata/abstracts only until separately approved. No pooled meta-analysis, product
fixes, approval bypasses, commits or publication of research artifacts in this run.

## Artifact/status contract

```
<campaign>/run.json
           authorization.md
           summary.md
           findings.json
           majorana/{commands.jsonl,coverage-stats.log,evidence.csv,report.md,provenance.json,predictions.json}
           nickelates/{commands.jsonl,coverage-stats.log,evidence.csv,report.md,provenance.json}
           <topic>/{manifest.json,batch-state.json,results.jsonl,provenance/events.jsonl,provenance/runs/<run_id>.json,...}
```

Create `run.json` with these exact fields (JSON, not the explanatory shorthand):

- `campaign_id`: safe ID; `purpose: "exploratory_not_gold"`; `started_at`: UTC ISO timestamp.
- `environment`: nonempty `os`, `agent`, `model` strings.
- `binary`: `path`, 64-lowercase-hex `sha256`, `version: {version, commit, date}`.
- `git`: `revision` (40/64 hex), `dirty` (boolean), `diff_sha256` (64 hex).
- `authorized_ceilings`: keys `majorana`, `nickelates`, each an object with
  `search_calls` (100/50 or lower authorized cap), `citation_seeds` (≤3),
  `citation_depth` (≤1), `citation_max_records` (≤30), `citation_max_api_calls` (≤6),
  `citation_retry_budget` (0), `paper_resolutions` (≤5),
  `process_timeout_seconds` (≤120), `search_resumes` (≤1). Zero disables a stage.
  Reduce command flags/timeout to any lower authorized limits before use; do not
  pass zero `--max-requests` (unsupported): skip search entirely when disallowed.
- `topics`: both IDs, each with `status` (`complete`, `partial`, `blocked`, `failed`),
  nonempty `reason`, boolean `search_attempted` matching the presence of attempted
  search commands, `stop_reason` (engine enum above, or `not_started`), `coverage` string (`unknown: reason` allowed), `gaps` string
  array, `usage`, `usage_unknowns`, and `engine_absence_reason` when needed.
  `usage` has `search_calls`, `citation_seeds`, `paper_resolutions`, `search_resumes`,
  `http_attempts`: nonnegative integers or null. Each null requires a nonempty
  reason under the same key in `usage_unknowns`. Known capped usage cannot exceed
  authorization. HTTP attempts have no invented count or equivalence to search calls.

A failed-before-search topic may lack engine-created files only with non-complete
status, explicit `engine_absence_reason`, and failed search/preflight command
receipts explaining the absence. Host files remain required: use honest empty
matrices, unsupported prediction slots, unknown stats and draft/gap reports.
Retain whatever engine files actually exist. Never manufacture receipts to pass.
A complete topic must have attempted search, engine artifacts and referenced runs;
`interrupted`, `error_threshold` and `not_started` are not complete. A structurally
complete topic may still have scientific gaps; a partial topic is not finished
research. If neither topic was attempted, the live task remains open.

### Provenance index and actual engine validation

Follow schema-v2 in the repository skill. Each topic's `provenance.json` has
`schema_version: "2"`, `provenance_level: "agent_reported"` (or `unverified`),
`question`, case `depth`, `sources` string array, UTC `timestamp`, structured
`rforge_version`, `errors` string array, `runs` of **actual** local IDs (or `[]`),
`entries` for host synthesis (`agent_reported`/`unverified` only), and
`outputs: [{"path": "report.md", "sha256": "<actual sha256>"}, ...]`.
The checker requires report, evidence, predictions (Majorana) and coverage stats
in the index, plus manifest, checkpoint, results and queries whenever present.
Include other saved outputs too; never index the provenance file itself or
mutable command/validator logs (their paths remain in command receipts). Manual outputs are topic-relative.
Keep immutable receipts/events unchanged. The checker verifies shape, safe paths
and references, **not** checksum equality, receipt internals or scientific support.

After finishing the index, for **each** topic:

```sh
record validate-1 provenance_validate "$RF" --json provenance validate "$RUN/$TOPIC/provenance.json"
```

Record the real validator result including failure. Do not overwrite validator
logs referenced by the index (use a new command ID if revalidation is needed).
A failed engine validation stays visible; structural acceptance does not turn it
into a pass. Host synthesis must never declare `engine_verified`; engine-only
receipts retain their real levels and are checked by the real validator.
Mechanical source identity/content/quote facets, agent semantic judgment and named
human approval remain independent; none asserts scientific truth.

### Findings and summary

`findings.json` is an array; **empty is valid**. Each object has:

- unique stable `id`; `topics` array of case IDs;
- `category`: `engine_bug_candidate`, `agent_error`, `upstream_failure`,
  `usability_gap` or `scientific_gap`; `severity`: `low`, `medium`, `high`, `critical`;
- nonempty `observed`, `expected`, `proposed_next_action`;
- `command_refs`: nonempty array of `<topic>/<command-id>` matching command records;
  `artifact_refs`: nonempty campaign-relative existing file paths;
- `reproduction_steps`: nonempty string array;
- `reproducibility`: `unconfirmed` or `offline_reproduced`. For an offline-reproduced
  engine bug candidate, also supply nonempty `offline_reproduction_artifacts`
  with existing reduced offline repro files. Do not call it confirmed without that
  evidence; syntax checking does not execute the reproduction. No quota of findings.

`summary.md` states **exploratory, not gold**; use sections `Verified failures`,
`Unconfirmed opportunities`, `Gates`, `Unsupported outputs`. Rank verified failures
separately from unconfirmed opportunities and state all unsupported/gated stages.
If any topic is non-complete include **`Status: partial`** and carry that status
into the final handoff. State scientific incompleteness independently of mechanical
completeness. Use the five suggested audit dimensions with observed counts and
source-linked explanations when meaningful; 30/25/20/15/10 weights are only a
proposal, not a weighted pass mark. Recall and mathematical/statistical correctness
remain not-assessed without an independent denominator/derivation. Never renormalize
missing scores into a flattering total.

Finally (offline):

```sh
python3 docs/benchmark/frontier-science/check.py --cases "$CASES" --run "$RUN"
```

Hand back exact commands, failures, usage/unknowns, gates, unsupported outputs and
ranked findings. Compare operational failure modes across topics, **not physical
measurements**. Preserve the tested snapshot. Triage first; any reduced-repro bug
fix requires a later scoped red–green–refactor task, not an in-run patch or retry loop.
