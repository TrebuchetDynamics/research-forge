# Verification facets and provenance levels

status: accepted (2026-10-19)

Agents increasingly hand ResearchForge claims and provenance they produced themselves. A single
"verified" flag would let a mechanical check (a URL resolves, a quote appears in a file) read as
scientific validation, and would let hand-written provenance pass as engine output.

**Decision.** Provenance records carry exactly one level: `engine_verified` (rforge produced the
outputs and their checksums recompute against an engine provenance event), `agent_reported` (a host
agent or human wrote it), or `unverified` (no claim, or legacy schema). The level is computed by the
validator; a record cannot raise its own level. Claim records carry five independent facets, never
collapsed into one status: **source identity**, **content integrity**, **passage match**, **semantic
support** and **human approval**. rforge may set the first three as engine-checked; semantic support
is always agent-reported with agent/model identity; human approval is set only by a named reviewer.
No facet, alone or combined, asserts that a conclusion is true.

## Considered options

- **Separate facets and computed levels (this decision).** Precise, auditable, costs a schema bump.
- **Single verified boolean.** Simple, but conflates mechanical and scientific checks.
- **Trust the writer's declared level.** No recomputation cost, but makes the level meaningless.

## Consequences

- `provenance.json` schema v2 adds the level and engine run linkage; v1 reads as `unverified`.
- Reports, packages, MCP responses and the web UI show facets/levels as stored, with wording that
  never implies truth.
- Passage match requires content hashes and stable passage identity; that work precedes claim checks.
