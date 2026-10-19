# Claim-evidence schema v0

## Purpose and boundary

`internal/claims` defines the storage-independent claim record for the
Research-quality benchmark and, later, the claim store. Start with
[`claims.Claim`](../internal/claims/claims.go). The governing decisions are
[ADR 0009](adr/0009-verification-facets-and-provenance-levels.md) and the
[verifiable research engine plan](verifiable-engine-plan.md).

V0 supplies Go types, JSON encoding and guarded assessment construction. The
append-only claim ledger and CLI live in `internal/claims` and
`internal/cli/claim_command.go`. V-D1 supplies stable passage hashes; V-D3 adds
[offline mechanical checks](../internal/claims/check.go), quoted-text results in
citation trace views, and [citation scoring](../internal/benchmarks/citation_correctness.go).
No LLM calls or export guards are implemented here.

## Record and evidence

| JSON field | Go type | Meaning |
| --- | --- | --- |
| `schemaVersion` | `string` | `"0"` (`claims.SchemaVersion`); distinct from legacy schema `"1"`. |
| `id` | `string` | Stable assertion ID within the originating research project or benchmark artifact. |
| `text` | `string` | Assertion, not a truth label. |
| `workflowStatus` | `string`, optional | Legacy suggested/accepted/rejected/corrected workflow state; never a facet outcome. |
| `evidence` | `[]EvidenceReference` | Supporting, contradicting or contextual references; an empty array is valid for an unassessed draft. |
| `facets` | `VerificationFacets` | Five independent, nullable assessments. |

Each `EvidenceReference` contains:

- `identifier: {scheme, value}`: source identity, e.g. `doi`, `arxiv`, `pmid`,
  `openalex` or `local`. Preserve the source's normalized identifier. A local ID
  is project-scoped; do not infer its scheme from its spelling.
- `sourceMetadata` (optional): expected bibliographic `{title, year}`. Nonempty
  values are compared with the resolved library record; missing comparison data
  is inconclusive, not a match. Parsed-document titles also provide comparisons.
- `passageId` (optional): identity of the cited passage, not its ordinal position.
- `relation`: `supports`, `contradicts` or `context`. This is a supplied
  characterization of the citation, not an engine judgment.
- `exactText` (optional): the quoted support text, retained without rewriting.
- `contentHash` (optional): `sha256:<64 lowercase hexadecimal digits>` over the
  complete stored passage's UTF-8 text, not the quote, PDF bytes or identifier.
  No extra quote normalization is applied before hashing; parser normalization
  is part of the versioned stored text. `parsing.Passage.ContentHash` stores this
  digest. `EnrichParsedDocumentModel` recomputes it without changing parser-assigned
  passage IDs, so unchanged reparses retain their identity and hash. IDs alone do
  not detect edits or guarantee identity across paragraph insertions/reordering.
- `offsets` (optional): `{start, end, parserName, parserVersion}`. A half-open,
  zero-based UTF-8 byte range `[start, end)` in the parser's document text stream,
  locating the cited passage (not the quote). Require `0 <= start <= end`.
  The parser version identifies the offset convention; these are not PDF byte
  offsets or Unicode character indices. A passage hash accompanies reusable
  offsets. Do not transplant offsets between parser versions or changed content.

[`parsing.ParsedDocument.ContentHash`](../internal/parsing/parsing.go) uses the
same prefixed SHA-256 format over its normalized UTF-8 document text stream:
title, abstract, each section heading and its passages, then each reference's
display text (first nonblank title, raw text or DOI, trimmed). Fields, including
empty ones, are separated by one newline, with no trailing newline. This is the
text stream used by parser offsets; IDs, parser versions, warnings, confidence
and annotations are excluded. Legacy parsed JSON without hashes remains readable,
but missing hashes do not establish content integrity.

The identity tuple is source identifier + passage ID + passage content hash.
Offsets are versioned location hints, not passage identity. Re-parsing unchanged
content should preserve passage identity; changed text requires rechecking the
reference and its facets. Legacy references can lack hashes, offsets or even a
passage ID; absence is not success. Non-passage legacy supports remain source-only
references until a future extension can identify their table/figure/etc. explicitly.

These plain data types do not validate source schemes, reference completeness,
range bounds, hash syntax or schema versions on JSON decoding. Those checks belong
at the future ingestion/checking boundary. In particular, providing a hash is not
proof that it recomputes.

## Five independent verification facets

The five JSON keys are always emitted. A `null` facet means **not checked**:
there is no fabricated checker or timestamp. A recorded assessment has:

```json
{
  "state": "passed",
  "checker": {"kind": "engine", "id": "rforge", "version": "dev"},
  "timestamp": "2026-10-19T12:00:00Z"
}
```

`state` is `passed`, `failed` or `inconclusive` (attempted but unable to decide).
`timestamp` is the time the assessment was made, encoded by Go's `time.Time` as
RFC3339 with optional fractional seconds; callers supply it, not an implicit clock.
`checker` carries kind (`engine`, `agent`, `human`), a nonblank actor ID, optional
software/agent `version`, and optional `model` (include the model version in its
identity when available).

| Facet | Question answered | Attribution |
| --- | --- | --- |
| `sourceIdentity` | Does the identifier resolve to the intended source and matching metadata? | May be engine-checked. |
| `contentIntegrity` | Does stored passage content recompute to the expected hash? | May be engine-checked. |
| `passageMatch` | Does the quoted text occur in the referenced passage under the declared matching policy? | May be engine-checked. |
| `semanticSupport` | Does the agent judge that the evidence bears the stated relation to the assertion? | Always agent-reported; named agent **and model** required. |
| `humanApproval` | Did the named reviewer approve this assertion and evidence? | Named human only; `failed` means declined, not scientific falsification. |

Each assessment covers this exact claim text and **all** listed evidence
references. A mechanical facet may pass only when every reference meets that
facet's criterion; any failure means failed, and missing/uncheckable references
prevent a pass. Do not carry assessments across edits to the text or evidence.
Per-reference mixed results are not represented by v0; a later detailed-check
extension can add them without replacing the record or these five fields.

There is no overall verification state, combined success flag or truth label.
Source identity can pass while passage match fails. A passage match says nothing
about semantic support. Agent-reported semantic support or a human approval does
not establish the truth of a conclusion. Workflow acceptance also proves none of
these properties.

### Construction and trust

The first three fields use `*Assessment`: their data may come from legacy or
external artifacts. A declared `checker.kind: "engine"` is attribution only, not
proof of an engine run. Computed provenance levels and checksum/run-event
validation are separate under ADR 0009; no caller-supplied level exists here.

`NewSemanticSupport(Assessment)` requires kind `agent`, a nonblank agent ID and
model identity, a supported state and a nonzero timestamp. The resulting
`SemanticSupport` has private fields. Its `Assessment()` method returns a copy;
callers cannot mutate it into an engine assessment. Its JSON decoder applies the
same constructor, including when nested in a `Claim`. Its attribution is always
`agent_reported`; it cannot promote a run's provenance level.

`NewHumanApproval(Assessment)` similarly requires kind `human`, a named reviewer,
a supported state and nonzero timestamp. JSON decoding enforces the same rule.
The zero values of these two guarded types are uninitialized and cannot be JSON
encoded. Use a nil pointer for an unchecked facet. Marshal also validates the
guarded value. Neither wrapper authenticates a person or an agent: this protects
against accidental mislabeling, not forgery.

Example construction (the caller supplies `at` and the assessment's real origin):

```go
semantic, err := claims.NewSemanticSupport(claims.Assessment{
    State: claims.StatePassed,
    Checker: claims.Checker{Kind: claims.CheckerAgent, ID: "pi", Model: "model-1"},
    Timestamp: at,
})
if err != nil {
    return err
}
record := claims.Claim{
    SchemaVersion: claims.SchemaVersion,
    ID: "claim-1", Text: "Artificial photosynthesis efficiency increased.",
    Evidence: []claims.EvidenceReference{{
        Identifier: claims.Identifier{Scheme: "local", Value: "paper-1"},
        PassageID: "p1", Relation: claims.RelationSupports,
        ExactText: "Efficiency increased.",
    }},
    Facets: claims.VerificationFacets{SemanticSupport: &semantic},
}
```

This illustrates construction only; an agent must actually assess the supplied
evidence before recording an outcome. Mechanical checks are not inferred from it.

## Mapping existing models

These are projection rules, not newly wired adapters or automatic upgrades. Keep
legacy artifacts available for fields that are outside the v0 projection.

| Existing model | V0 projection | What must not be inferred |
| --- | --- | --- |
| [`evidence.CitationLockedSuggestion`](../internal/evidence/citation_locked_suggestions.go) | `ID` → `id`; `SuggestedText` → `text`; `Status` → `workflowStatus`. Each `CitationLocks` entry maps `PaperID` to the source identifier, `Ref` to `passageId` only if it is a passage reference, and `ExactText` to `exactText`; characterize as supplied `supports`. | Model name/version describes the generator, not a semantic assessment. `ReviewerDecision` has no timestamp, so it cannot alone construct a human facet. Nonempty locks do not prove passage match. Preserve prompt/kind/reviewer note in the original artifact. |
| [`report.ClaimTraceView`](../internal/report/citation_trace.go) | `ClaimID`, `ClaimText`, `ClaimStatus` map to the same three fields above. `PaperID` identifies the source. Each `TracePassage.PassageID` identifies a candidate reference; parser name/version and `Offset.Start/End` map to `offsets` when complete. Join the original lock to recover the actual quote. | Trace passage text is the entire passage, not necessarily the quote. There is no content hash yet: leave it absent until recomputed. `AcceptedEvidence` is joined by paper ID, not proof of claim-specific support. Analysis rows, API records and PDF links remain trace-view details. All facets remain unchecked without separate check records. |
| Knowledge-graph [`report_claim` nodes](../internal/knowledge/graph.go) (`builder.addClaim`) | Node `claim:<ClaimID>` maps back to claim ID within its project; label maps to text, `status` property to workflow status. `claims_about` links identify papers; `supported_by` edges identify candidate evidence to resolve back to a passage. | Those edges currently select paper-wide accepted evidence. An edge is neither an exact quote match nor semantic support. Graph nodes carry no checker/time/hash; none can be manufactured from an edge. Preserve v0 claim IDs when projecting back into the graph. |

For legacy opaque `PaperID`, use `{scheme: "local", value: PaperID}` unless the
source library record establishes the identifier scheme. Preserve missing data as
missing; do not invent a check time from import time. An accepted legacy claim can
therefore legitimately have five unchecked facets.

## Deterministic checks

`rforge claim check --project <path> --id <id> [--json]` loads the latest claim,
`data/library.json` and `parsed/*.json` offline. It appends a new claim snapshot
and returns the five separate facets plus ordered per-reference check details.
A completed check exits successfully even when a facet fails; consumers must
inspect the facet states. Read/parse/write errors fail the command.

Source identifiers resolve only within the supplied library, including `local`
identifiers matching `PaperRecord.RecordID`. At least one comparable expected or
parsed title/year is required to pass identity; any disagreement fails it.
Content integrity recomputes SHA-256 from the complete stored passage bytes and
compares both the reference and stored passage hashes. Missing hashes or
ambiguous/unavailable passages are inconclusive, not success.

[Passage matching](../internal/claims/passage_match.go) first selects the first
exact quote occurrence. Failing that, it uses Unicode 15.0 canonical NFD
normalization and collapsed/trimmed Unicode whitespace, without case folding,
compatibility folding or accent removal. Generated Unicode tables keep this
offline and dependency-free. A match cannot split an original character's
canonical decomposition. Results include half-open original passage UTF-8 byte
ranges and, when parser metadata/ranges are available, absolute document quote
`offsets`. These **returned quote offsets** differ from the reference's **input
passage offsets**. Supplied passage offsets must match the stored parser/range.

Each deterministic facet passes only if all references pass; failure dominates
inconclusive. Empty evidence is inconclusive. Semantic support and human approval
are preserved without inference. The Go check/scoring services require an explicit
check timestamp; the CLI supplies the current time.

`CitationLockedSuggestion.CheckCitationLocks` compares each `ExactText` to its
stored passage (plain passage ID or `paperId:passageId`), and
`BuildCitationEvidenceTraceView` exposes these ordered results as
`citationMatches`. Existing passage links and review status remain navigation and
workflow data, not proof of quote matching or semantic support.

## Research-quality benchmark contract

Citation artifacts use `[]claims.Claim` directly (including `EvidenceReference`,
`VerificationFacets`, `Assessment`, `SemanticSupport` and `HumanApproval`), not
benchmark-specific copies of their JSON shape. The benchmark envelope may add
question/set IDs, set checksum, engine version and metric counts. It must not
rename facets, inject fake engine assessments or replace missing assessments with
passing values.

[`benchmarks.ScoreCitations`](../internal/benchmarks/citation_correctness.go)
consumes these records and the caller's locked evidence corpus, recomputing the
three deterministic facets for **every citation occurrence**, including duplicate
citations and missing/uncheckable references. Supplied facet states are not trusted.
The numerator counts occurrences passing all three mechanical checks; the
explicit denominator includes every occurrence. Per-facet passed/failed/inconclusive
counts and the subset missing required reference fields are reported separately,
with each checked occurrence retained in a private audit. `citation_status` is
`measured` for nonempty citations, or `no_citations` with 0/0 and a null value.
Semantic support and human approval remain separate and are never inferred.
The old [`cross_tool.go`](../internal/benchmarks/cross_tool.go) demo is unchanged;
this service is the citation-scoring seam for the research-quality harness, not
synthetic evidence or a new improvement claim.

The claim store appends the same records at `data/claims.jsonl` without renaming
fields or migrating benchmark evidence. Store sequence/run linkage can be
additive; there is no automatic legacy conversion.

## Validation and update triggers

Run `go test ./internal/claims` for populated/unassessed JSON round trips and
constructor/JSON checker-policy tests. These tests are offline with fixed times.
Update this page when V-D1 fixes the parser/hash representation, V-D2 adds the
store, V-D3 adds real facet checks/scoring, or any legacy model mapping changes.
