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

Keep protocols, schedules, raw results and condition mappings with the custodian.
The scorer receives only blinded answers and locked gold, then freezes scores
before conditions and operational counters are joined.
