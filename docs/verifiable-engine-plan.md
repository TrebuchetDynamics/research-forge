# Verifiable research engine plan

Status: accepted direction, 2026-10-19. Tasks live in [TODO.md §12](../TODO.md#12-verifiable-research-engine-cycle).
Origin: competitive review of Feynman (Companion-Inc) and related research agents, refined in a
plan-with-me session with the owner.

## Brief

**Goal.** Prove, with a held-out benchmark, that a Feynman-class agent produces better research when
equipped with rforge than without it. Feature parity with Feynman is not the goal.

**Users.** Host agents (Claude Code, Codex, Pi, others) and the humans who review their output.

**Problem.** Agents that use rforge today get an aggressive default search depth, hand-written
provenance that looks identical to engine-written provenance, a stale install requirement, no typed
tool interface, and no way to show whether rforge improves their research. ResearchForge's
deterministic checks link claims to evidence by ID only; quoted text is never compared with stored
content.

**Success.** A published baseline from held-out questions, then measured deltas for each later
change; agent outputs carry verification facets that state exactly what rforge checked.

## Owner decisions (2026-10-19)

1. **Engine, not agent.** rforge stays agent-independent. No built-in LLM orchestration or subagents;
   semantic judgment belongs to the host agent and is recorded as agent-reported.
2. **Standard depth by default.** Comprehensive only when coverage signals say results are thin or the
   user asks. Every run reports hard caps and a stop reason.
3. **Provenance levels.** `engine_verified` / `agent_reported` / `unverified`. Reports and packages never
   present a lower level as `engine_verified`; hand-written fallbacks must declare `agent_reported`.
4. **In-progress commands are the owner's.** `paper`, `rank`, `brief`, `compare`, `lit`, `note`,
   `history`, `replicate`, `draft` and batch checkpoints are in development; this plan builds on them
   after they land and evaluates them before adding overlapping functionality.
5. **Deferred.** Semantic ranking, learned source routing, persistent cross-run memory, GitHub
   description/topics (owner chore).
6. **Benchmark before improvement claims.** Held-out questions, expected sources, retrieval recall,
   citation correctness, cost and latency form a baseline; improvements are reported as deltas.
7. **`engine_verified` is narrow.** It means rforge independently checked specific evidence
   properties, never that a conclusion is true. Facets stay separate: source identity, content
   integrity, passage match, semantic support, human approval. See
   [ADR 0009](adr/0009-verification-facets-and-provenance-levels.md).
8. **MCP is a thin adapter.** CLI, MCP and the skill share the same Go services and persistence. No
   MCP-only research logic or MCP-specific results.
9. **Order A → B → C → D**, with the claim-evidence schema designed during B so benchmark artifacts
   become evidence-graph records without migration.

## Planner's calls (owner may override)

- **Coverage signal** for stop conditions is marginal yield: new unique deduped records per query.
  Saturation = yield below a threshold for N consecutive queries. It is deterministic and needs no LLM.
- **Two benchmark tiers.** *Engine tier*: offline replay of recorded HTTP snapshots, deterministic,
  runs in CI. *Agent tier*: live, opt-in, repeated runs with confidence intervals; scored with the
  same metrics. Hardcoded `DefaultCrossToolBenchmarkInput` numbers are retired or labeled as demo.
- **Gold sets from independent ground truth.** Preferred source: included-study lists of published
  systematic reviews (an outside party decided relevance), plus the artificial photosynthesis fixture
  topic. Held-out split is checksum-locked and never used for tuning.
- **`engine_verified` cannot be typed by hand.** It is accepted only when output checksums recompute
  and the run ID matches an engine provenance event; otherwise the validator downgrades or fails.
  Schema v1 `provenance.json` reads as `unverified` with a warning.
- **Semantic support is always `agent_reported`** (with model/agent identity). rforge never runs an LLM.
- **MCP v1 exposes read/run tools only.** Screening approval, claim acceptance and package export stay
  CLI/human-gated. Transport: stdio first, official Go MCP SDK
  (`github.com/modelcontextprotocol/go-sdk`), which is the first new direct dependency since playwright.
- **Service extraction precedes MCP.** Logic now inline in `internal/cli` (search batch/resume, rank,
  brief, compare, lit, history) moves into service packages with CLI parity tests.

## Evidence (verified 2026-10-19)

- `go.mod` requires Go 1.26 / toolchain 1.26.6; `skills/research-forge/SKILL.md:85` says Go 1.22+.
- `SKILL.md:69` makes Comprehensive (20–30 query variants) the default; `:132`, `:442` allow manual
  `provenance.json`.
- No MCP code or dependency. `internal/jobs` is a stub (`jobs.go:5-8`); resumable state lives in
  `internal/cli/search_batch_state.go`.
- `internal/benchmarks/cross_tool.go:42-55` reports hardcoded constants; no gold sets or qrels exist.
  `retrieval.DefaultRetrievalBenchmarkFixture` (3 synthetic queries + checksum) is the closest format.
- `report.BuildCitationEvidenceTraceView` links by paper/passage ID only; `CitationLockedSuggestion`
  `ExactText` is checked non-empty, never matched. Parsed text and passages have no content hash;
  passage offsets are parser-version-specific. Document files do have SHA-256.
- Provenance path mismatch: writers use `provenance/events.jsonl`; `reviewpkg`, `webui`, `protocol`
  read `data/provenance.jsonl`.
- In-progress commands: logic in `internal/cli` except `sources.ResolvePaper`; no provenance events;
  `rank --out` drops rankings and overwrites any `provenance.json` in the target dir (fails the
  validator); Unpaywall email hard-coded; `replicate`/`lit` help claims "deterministic" but outputs
  include timestamps/live data; `rank`, `draft`, `paper` have no CLI tests. `rank` score has no
  query-relevance term.

## Non-goals

- Agent orchestration, subagents, chat UI, or LLM calls inside rforge.
- New connectors for their own sake.
- Items in decision 5.

## Anti-goals (failure even if it ships)

- Any path that labels agent-typed or manual data `engine_verified`.
- `engine_verified` wording that implies a claim is true.
- MCP results that differ from CLI results for the same inputs.
- Improvement claims without a benchmark delta against the locked baseline.
- Tuning on the held-out set.

## Acceptance

- Docs parity check fails CI when skill/README/installer disagree with `go.mod`.
- Search runs report caps, coverage and a structured stop reason; skill default is Standard.
- `rforge provenance validate` reports a level and cannot be satisfied by a hand-typed `engine_verified`.
- `rforge benchmark run` produces a reproducible engine-tier report on a checksum-locked held-out set;
  a dated baseline is committed.
- `rforge mcp serve` tools return the same JSON as their CLI `--json` equivalents on fixtures.
- `rforge claim check` reports the five facets separately; report export guards use them.

- **Release gate (owner, 2026-10-19):** ResearchForge 2.0 must demonstrably improve evidence quality,
  retrieval effectiveness or research efficiency over the same agent without rforge (TODO V-R1).

## Owner answers (2026-10-19, follow-up)

- **Threat model:** guard against accidental mislabeling only; no signing (signed log deferred).
- **Gold sets:** included-study lists of published systematic reviews, with leakage controls: the
  reference review is excluded from retrieval and expansion seeds, searches are date-capped at its search
  date, and agents never see it.
- **Agent tier:** Pi primary, Codex secondary; same model per pair; pilot 8 questions × 2 conditions ×
  3 repetitions (48 Pi runs); equal token and search budgets; blind scoring.
- **V0.1:** the builder commits the in-progress commands selectively; implementation and planning
  changes go in separate commits.

## Open questions

None blocking. The pilot's n = 8 questions is for estimating effect size; the release gate is judged on
the expanded set.
