# RF-BENCH — offline planning round 2

Status: Planning stopped after round 2 of 10; no new building batch selected.

## Brief and boundaries

[V0-018-8f442e1d](../../tasks/items/V0-018-8f442e1d.md) asks for a bounded review of round 1, including the citation-corpus gap, not a backlog sweep or implementation. The [accepted engine plan](../verifiable-engine-plan.md#verifiable-research-engine-plan), [benchmark specification](../benchmark/spec.md#6-scoring-definitions-shared-by-both-tiers) and [RF-BENCH goal](../../tasks/goals/RF-BENCH.md) remain authoritative. The goal is research-quality measurement against held-out ground truth, not more benchmark tooling for its own sake.

Success for this planning pass is an evidence-backed disposition of both completed lanes and the remaining proof. Invariants: no live calls, credentials, spending, new registration, changed gold/snapshots, historical artifact rewrite, held-out tuning, invented receipt or goal qualification. Preserve V0-012 and all owner live deferrals. Manual planning closure is not coordinator verification.

## Evidence reviewed

Review baseline: `c871287` (the verified close of V0-017). Both round-1 task records are already done with coordinator receipts; neither needs reopening or a replacement task.

- **V0-016-db12f60e:** execution-common `slot-data/V0-016-db12f60e/attempt-1/report.json`, candidate `bb1ae5c60a7478c08d925454908ef3cc0c4a8b92`, records passing comparison, rejection, command and owning-package checks. Current `internal/benchmarks/comparison.go` validates compatible replay identities and question matrices, recomputes aggregates and contributing populations, and subtracts candidate minus baseline with null preservation. `benchmarks/report-delta/main.go` uses strict decoding and canonical stdout without writing inputs. Its limitations explicitly exclude replay-speed, custody and superiority claims. Outcome A is delivered, not a new research-quality win.
- **V0-017-28c122bb:** execution-common `slot-data/V0-017-28c122bb/attempt-1/report.json`, candidate `c2cba1f3aab9a58dd75401a4211ebb4cbeeaa008`, records passing public readback, 27 corruption subcases and both owning packages. Current `pilot_readback_test.go` and `pilot_readback_helpers_test.go` rescore/freeze, check the 48-slot matrix and public digest links, recompute summaries/contrasts/sizing, and preserve outage, budget-stop and unavailable-facet classifications. Outcome B is delivered. Public readback cannot reconstruct private mappings/billing or qualify historical host isolation.
- **Citation gap:** `internal/benchmarks/citation_correctness.go` already provides `ScoreCitations` over stored claims and an explicit `claims.CheckInput`; its existing occurrence test covers repeated, missing, unresolved and forged-facet inputs. In contrast, `benchmarks/agent-tier/protocol.go` accepts only `citation_metric_version = not_implemented`, and its `Claim` contains only text, quote and locator. `blind.go` scores bibliography retrieval, not typed claim-to-source occurrences. The [runbook](../benchmark/agent-tier.md#6-metrics-and-interpretation) explicitly says a pinned passage corpus/facet adapter and a newly registered scorer are missing. The committed pool-a manifest and metadata snapshots do not supply that corpus, and the public pilot does not supply per-occurrence passage IDs/content hashes. Existing synthetic checker fixtures are not independent evaluation ground truth.
- **Remaining proof and authority:** RF-BENCH is still unmet. Its recorded deterministic engine capture/replay evidence and round-1 infrastructure do not establish an agent-quality gain. [V0-012-88342f33](../../tasks/items/V0-012-88342f33.md) already owns baseline provider repair and remains owner-deferred for Brave credentials or SearXNG access. The owner manually deferred V-B6-codex and the release/live campaign rows on 2026-10-22 without claiming live qualification. The accepted plan requires a separately registered expanded experiment for the release conclusion; the published pilot is not that proof.

Execution-common paths above mean `/home/xel/git/research-forge/.git/autowork-execution/`. Reports were read in place, not copied or made required worker inputs. Passing worker checks are historical reported evidence; this pass does not manufacture a new execution receipt.

## Planner's call and alternatives

**Planner's call:** stop this rolling chain here rather than manufacture a third offline-tooling batch. Round 1's two known, input-backed slices are delivered. This is a bounded planning judgment, not a claim that no useful offline work could ever exist.

- Repeating comparison/readback work would duplicate completed contracts.
- A new synthetic citation utility could be engineered offline, but `ScoreCitations` already checks the core metric. Another wrapper or synthetic score alone would not provide the absent independent citation evidence or advance the goal's remaining empirical proof sufficiently to justify this batch.
- Retrofitting citations into the public pilot would change its registered measurement contract and invent unavailable source-to-occurrence evidence. Reject that route; unavailable citation values remain null.
- A prospective corpus-backed scorer is possible future work, not inherently dependent on baseline provider access. It would need a defined lawful evaluation corpus and explicit occurrence inputs plus a new scorer/protocol, none of which this pass treats as already supplied or approved. No corpus access, scientific labels or owner choice is invented. This stop does not make citation scoring mandatory for every possible route to the goal.
- Provider repair and a fresh expanded live campaign already have owner/access boundaries. Do not duplicate V0-012, reopen deferred tasks, select a substitute baseline or authorize spend to keep the queue busy.

The continuation question below concerns which further outcome, if any, the owner wants scoped. It does not demand a coordinator handoff or forbid an unattended builder from repairing an existing defect in a future task's own checks.

## Stop disposition and continuation

This round adds **one** RF-BENCH Needs decision record, [V0-019-ea266b81](../../tasks/items/V0-019-ea266b81.md), no building task and **no successor planning task**, even though 2 is below the preserved limit of 10. Close V0-018 manually in this same planning commit. Keep RF-BENCH unmet and V0-012 byte-identical.

Canonical stop question: [Q-V0-018-8f442e1d](../../BLOCKERS.md#q-v0-018-8f442e1d--planning-stopped).

**Question:** For unmet RF-BENCH, should planning remain paused under the existing provider/live deferrals, or which further offline benchmark outcome should be scoped next?

An answer may authorize a new planning slice; it does not itself provide credentials, approve data rights, register an experiment or authorize a live campaign. No runner is started or restarted by this pass. There is no automatic round-3 continuation to stop.

## Owner decision: planning remains paused

On 2026-10-23 the owner selected **option 1: keep further offline benchmark planning paused**. This resolves Q-V0-018-8f442e1d and the decision-only task V0-019-ea266b81; it is not completion or scientific qualification of RF-BENCH.

Keep RF-BENCH **unmet**, save no new building work or successor planning round, and preserve the V0-012 provider-input deferral and all existing live/release deferrals. No new execution, provider access, credentials, evaluation corpus, protocol change, spending, installation or publication is authorized. Resume future planning only on a new explicit owner instruction. Manually closing the answered decision record writes no coordinator receipt or verified research evidence.

## Validation and readiness

The new record is an owner continuation question, deliberately not executable building work. There are no new parallel write lanes or unmet worker inputs to conceal. The preserved V0-012 is still waiting for its existing owner-supplied input, not for this question.

Executed closeout checks:

- Fresh-context, read-only builder/plan review: **nothing important left**. It independently checked the finished reports and citation gap, found no invented owner gate, and confirmed that prospective corpus work is not inherently dependent on provider repair.
- `autowork tasks validate --worktree .`: **pass**, 73 tasks, 7 goals, no drafts.
- Local link/anchor and disposition check: **pass**, 12 links/anchors resolve; V0-012 and the RF-BENCH goal remain byte-identical to the review baseline; no open successor plan exists and the new continuation record is owner-only in Needs decision.
- `make check`: **pass**, exit 0; full offline Go/adapter tests, vet, completion audit, inventory and whitespace gate completed. The repository pre-commit hook also requires this full check when saving the planning commit.

No proposed product test or fixture is implemented in this planning pass. These checks validate the existing code and planning records; they do not establish live host/provider qualification or meet RF-BENCH.
