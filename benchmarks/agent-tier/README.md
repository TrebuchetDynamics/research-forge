# Agent-tier benchmark tooling

Start with the [protocol and runbook](../../docs/benchmark/agent-tier.md).
The normative contract is [benchmark spec §8](../../docs/benchmark/spec.md#8-agent-tier-paired-protocol-v-b6).

```sh
go build -o "$TMPDIR/agent-tier" ./benchmarks/agent-tier
go test ./benchmarks/agent-tier ./internal/benchmarks -count=1
```

Commands: `stub`, `prepare`, `blind`, `score`, `report`; all require `--out`.
Outputs are create-only. The stub is offline canned fixture data, not an
improvement claim. No command starts Pi/Codex or performs live research.

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

Keep protocols, schedules, raw results and condition mappings with the custodian.
The scorer receives only blinded answers and locked gold, then freezes scores
before conditions and operational counters are joined.
