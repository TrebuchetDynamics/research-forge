# Agent-tier benchmark tooling

Start with the [protocol and runbook](../../docs/benchmark/agent-tier.md).
The normative contract is [benchmark spec §8](../../docs/benchmark/spec.md#8-agent-tier-paired-protocol-v-b6).

```sh
go build -o "$TMPDIR/agent-tier" ./benchmarks/agent-tier
go test ./benchmarks/agent-tier ./internal/benchmarks -count=1
```

Commands: `stub`, `prepare`, `blind`, `score`, `report`; all require `--out`.
Outputs are create-only. The stub is offline canned fixture data, not an
improvement claim. None of these Go commands starts Pi/Codex or performs live research. The separate
[opt-in Pi adapter](live/README.md) implements the authorized Pi-only pilot;
[registration](pilot-pi-2026-10-21/registration.json) and
[pilot status/results](../../docs/benchmark/pilot-pi-2026-10-21.md) are separate
from this canned fixture.

Protocol v2 registers a provider mix (default Crossref, Semantic Scholar, arXiv;
no OpenAlex dependency), continuation on failure and per-observation quarantine
after the first 429. Both conditions require ordered provider receipts; import
rejects retries of quarantined providers. The qualified live gateway must enforce
this before connector-internal retries: `--continue-on-error` alone cannot.

`stub --rate-limit-provider crossref --set SET --out NEW_DIR` emits a canned 429,
its Retry-After and a subsequent skip in both conditions, continuing other
providers. Blind/score/report then preserves those failures in the joined
`provider_availability` matrix and flags both members of affected pairs, without
dropping or adjusting their scores. This offline fixture does not qualify a
live gateway; follow the runbook's separate outage-injection qualification.

## Offline published-pilot regression check

From the repository root:

```sh
go test ./benchmarks/agent-tier -run '^TestPublishedPilotReadback' -count=1
```

`TestPublishedPilotReadback` reads the committed
[pilot package](pilot-pi-2026-10-21/) and locked `pool-a` set without modifying
either. It rescores the public blinded answers to a temporary directory and
compares scores and freeze receipts byte-for-byte. Test-local guards verify
public canonical protocol/answer/score/report digest links, the registered
schedule and unique 8-question × 2-condition × 3-repetition matrix. They recompute
all overall/domain aggregates, paired contrasts and registered analysis sizing
from published rows. Floating comparisons use absolute tolerance **1e-9**;
integers, digests, booleans and nulls match exactly.

The guards retain all 48 rows, including 24 failed native observations/48 failed
native attempts, provider loss in every pair, 23 with-tools budget stops and
unavailable citation/dedupe facets. Availability is reconciled against public
provider events and counters; a `completed` native run is not successful retrieval.
`TestPublishedPilotReadbackRejectsTampering` deliberately corrupts temporary
copies to check rejection of missing/duplicate slots, changed digests, summaries,
sizing, availability flags, counters and fabricated unavailable measurements.
Both tests check that original package and locked-set bytes remain unchanged.

This lane needs only Go and committed public data: no Pi installation,
bubblewrap, credentials, billing input or network. It does not reconstruct the
private condition mapping, certify historical blinding/scorer isolation, verify
private billing or qualify a live host/provider. Reproducible arithmetic is not
evidence of scientific validity or superiority over functioning native search;
see the [pilot limitations](../../docs/benchmark/pilot-pi-2026-10-21.md).

Keep protocols, schedules, raw results and condition mappings with the custodian.
The scorer receives only blinded answers and locked gold, then freezes scores
before conditions and operational counters are joined.
