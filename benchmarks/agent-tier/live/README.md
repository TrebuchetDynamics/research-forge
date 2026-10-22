# Opt-in Pi pilot adapter

External benchmark tooling for [the agent-tier protocol](../../../docs/benchmark/agent-tier.md), not orchestration inside rforge. Live spending requires owner authorization and a committed registration. This adapter is Linux/Pi-specific; it does not qualify Codex.

## Boundaries

- `custodian.py` owns observations, the trusted gateway, private engine artifacts and actual OpenRouter billing receipts. It invokes the real `rforge search batch` through existing source endpoint overrides. Never mount this process's private directory into the agent or scorer.
- `gateway.py` permits Crossref, Semantic Scholar and Europe PMC only. It applies provider date filters and independent earliest-publication-date upper bounds before returning records, checks all reference-review aliases before filtering and after rendered native extraction/engine merging, and counts every HTTP attempt/redirect. First 429 quarantines the canonical provider across queries. Errors returned to engine connectors use a non-retryable local 400; actual upstream statuses/Retry-After remain in receipts. A skipped call makes no upstream request and never waits.
- `pi-session.mjs` runs one actual Pi SDK session in bubblewrap with only its selected input, fresh writable workspace, pinned Pi distribution, original installed Search Hub module and gateway socket. A custom resource loader discovers no user/project instructions, skills or extensions. No file/shell/network tool is available. Network namespace is shared for the provider API, but all model-accessible research tools route exclusively through the gateway.
- With-tools exposes `rforge_search` and `submit_answer`, with the exact methodology skill embedded in registered condition instructions. Without-tools exposes the original Search Hub `web_search` and the same `submit_answer`, with keyless DuckDuckGo only. It has no rforge binary, skill, plugin, outputs or research cache. No Search Hub implementation is copied or replaced.
- Native raw search results are date-filtered before original parsing/formatting. Publication dates are independently obtained from the selected results' public-page citation metadata or URL-matched Article JSON-LD; unrelated JSON-LD dates are not evidence. Unknown/after-cap dates are excluded and counted. Cross-host redirects fail rather than introducing an unqualified provider. This conservative policy can reduce native coverage and must be disclosed. Opaque Jina `web_read` is absent in both conditions.
- Each model request pins the exact model, Anthropic upstream, temperature/top-p and at most 4,096 output tokens. Cached-read/write input counts toward the total cap. Missing/mismatched streamed token usage fails closed; native generation token counts and actual billed USD must reconcile before a raw row is accepted. Billing lookup retries only eventual HTTP 404 indexing, never a model or research request.
- Budget checks precede invocations. An indivisible response can overrun: retain actual counters and its already-produced structured answer, mark `budget_exhausted`, and do not execute further host tools. Unstructured partial text remains private, not converted into invented citations.

## Prerequisites and offline check

Python 3 (stdlib only), Node.js 22.19+, bubblewrap/user namespaces, Go from `go.mod`, an existing Pi **1.3.0** managed installation and its installed Search Hub module (the pinned module exports `registerSearchHub`). No new dependency manifest or global host settings are needed. Paths default to the normal personal Pi installation; override `PI_MANAGED_INSTALL_ROOT` or `PI_SEARCH_HUB` for another existing installation. The task's `TMPDIR` and reserved `AUTOWORK_PORT_BASE` must be set.

```sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover \
  -s benchmarks/agent-tier/live -p 'test_*.py'
node --check benchmarks/agent-tier/live/pi-session.mjs
go test ./benchmarks/agent-tier ./internal/benchmarks -count=1
```

The SDK integration test uses a local SSE provider, not credentials or a paid model. It verifies sandboxed output, measured cached-input accounting, missing-usage rejection and retention of an in-flight budget-stop answer. Gateway tests cover provider filters/raw dates, excluded aliases/rendered titles/redirect targets, HTTP caps and first-429 quarantine. These deterministic tests alone do **not** establish live qualification.

## Authorized execution

Use the committed [Pi registration](../pilot-pi-2026-10-21/registration.json) and [protocol](../pilot-pi-2026-10-21/protocol.json). Do not silently change prompts, model, loadout, provider mix, caps, questions or analysis. The registered shared task permits at most two research invocations; this is a bounded retrieval pilot, not an exhaustive systematic review. Pin/build the engine revision recorded there and verify its capabilities before execution.

```sh
work="$TMPDIR/pi-pilot" # custodian-only; use a fresh directory
# Build these first, retaining exact version/revision/binary digests.
go build -o "$work/agent-tier" ./benchmarks/agent-tier
# Build rforge with the registration's version/commit/date ldflags.
"$work/agent-tier" prepare --set benchmarks/sets/pool-a \
  --protocol benchmarks/agent-tier/pilot-pi-2026-10-21/protocol.json \
  --out "$work/pi-private"
# One scheduled pair, in original order; repeat start=2,4,...,46, never cherry-pick.
PYTHONDONTWRITEBYTECODE=1 python3 benchmarks/agent-tier/live/custodian.py \
  --private "$work/pi-private" --set benchmarks/sets/pool-a \
  --rforge "$work/rforge" --start 0 --count 2
```

The adapter resolves the existing OpenRouter API-key credential from personal Pi auth, never prints it, supplies it only to the isolated host runtime, and removes its temporary credential file on exit. It never changes personal auth/config. Keep the parent private directory owner-only. Observe the registration's USD execution cap in addition to per-observation caps; the custodian operator owns this aggregate spending gate.

Outputs are create-only. `run-NNNN/observation.json` contains the exact answer, private transcript, realized model/loadout and measured gateway events. `billing.json` contains only generation identity/token/cost fields. `raw-run.json` is written only after billing reconciles. A delayed billing read does not authorize a new agent run: finish reading receipts for the retained observation. Infrastructure replacement requires retaining superseded attempts and replacing **both** members of a pair, never a selective outcome retry. A protocol/leakage failure invalidates the experiment.

Collect all 48 `raw-run.json` files in schedule order into `experiment.json`, with `protocol_sha256` computed from canonical `private/protocol.json` bytes excluding its LF, plus the unchanged schedule and runs. The offline utility validates that matrix.

Before scoring this pilot, apply its deterministic schema projection to a **new private directory**:

```sh
PYTHONDONTWRITEBYTECODE=1 python3 benchmarks/agent-tier/live/extraction.py \
  --private "$work/pi-private" --out "$work/pi-projected-private"
```

`extraction.py` takes no gold, relevance labels or scores. It case-folds supported scheme labels, maps only DOI-shaped Crossref connector `CrossrefID` aliases to `doi`, and omits only empty-string identity placeholders. It preserves all nonempty identifier values, bibliography ranks (even empty ones), duplicates, claims and quotes, rejecting unknown schemes/non-DOI `CrossrefID` values rather than guessing. The create-only receipt records source/projected hashes and counts. Retain originals, failed prior bundles and all operational evidence; this is schema extraction, not a new agent run or changed relevance judgment.

Then follow runbook §5 using the projected private directory: blind answers, run `score` in a separate bubblewrap filesystem/PID/network namespace containing only a **statically built** utility (`CGO_ENABLED=0 go build`), locked set, blind answers and new score output, freeze, then `report` as custodian. Publish blinded answers, frozen scores/freeze digest, joined report/audit and measured analysis; retain mappings, transcripts, source bodies and identifiable billing receipts privately. The scorer's blind mount must contain only `answers.json`, never the entire published package (which also contains joined labels/counters).

The [completed 48-run record](../../../docs/benchmark/pilot-pi-2026-10-21.md) discloses native HTTP-202 search failures in every without-tools observation, source 429s and 23 with-tools token-budget stops. Do not retry those outcomes selectively, hide overruns, or interpret the positive contrast as superiority over functioning native search. Expanded execution needs a new approved set, protocol and spending authority.

See [the Pi pilot record](../../../docs/benchmark/pilot-pi-2026-10-21.md) for execution status, results and limitations. It is not a release-gate or Codex result.
