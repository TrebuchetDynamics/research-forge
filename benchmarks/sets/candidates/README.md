# Unlocked pilot candidates — AP fallback

[`pool-a/set.json`](pool-a/set.json) contains **8 held-out questions, 4 separate dev
questions and 113 included-paper records** from twelve independent published
systematic-review families. Held-out domains are clinical medicine, education,
social policy and performing arts. **Artificial photosynthesis (AP) is absent.**

This uses the **owner-authorized fallback of 2026-10-19**: if no AP review yields a
verifiable complete included-paper list, ship without AP and document the checked
reviews and their qualification failures. The [review sheet](pool-a/review.md#ap-fallback)
and [discovery record](discovery.md) record that decision and evidence. This is not
a claim to satisfy the specification's original mandatory AP coverage, nor a
relabelled clinical question or synthetic AP substitute.

The preferred 14-study TiO2 nanotube review was checked again under the new
month-precision rule. April 2021 is acceptable as `search_date_precision: "month"`
with cap **2021-04-01**; its date is no longer the rejection reason. Public table
snippets identified 13 distinct included-paper references, but did not establish
the complete corpus corresponding to the reported 14 studies. Neither those
snippets nor the 50-item bibliography were promoted to gold.

## Lifecycle and access separation

**Candidate, unapproved, unlocked, metadata only.** Approval and snapshot digest
remain null. No snapshots, benchmark execution, scores or baseline were recorded.
All twelve admitted reviews have day precision (the schema default when
`search_date_precision` is omitted); no gold was dropped (see [Dropped gold](pool-a/review.md#dropped-gold)).

All curation/provenance is `agent_reported`. The source identities, dates, rights,
leakage controls and unresolved curator questions are in the review sheet;
[provenance.json](provenance.json) distinguishes the inherited discovery batches
from this follow-up. No article text, abstracts, PDFs, API bodies or book-preview
images are redistributed. Never expose this directory, review identities, gold,
exclusions or discovery notes to benchmark agents or query planners.

Owner review and approval remain V-B4-lock work. Do not record, lock, execute or
publish a baseline from these candidate files. The unchanged engine execution
policy still requires AP for approved/locked evaluation sets; the fallback here
is **candidate-validation-only**, not permission to bypass that recording gate.
V-B4-lock must reconcile capture policy before recording if AP remains absent.

## Offline validation

Validate the original JSON against the embedded schema in
[specification §2](../../../docs/benchmark/spec.md), including date formats,
then use the candidate-only Go validation entry point:

```go
err := benchmarks.ValidateCandidateSet("benchmarks/sets/candidates/pool-a", true)
```

`true` explicitly opts into the documented owner AP fallback. It waives **only**
held-out AP presence; minimum question counts, at least three held-out domains,
source identities, date caps, leakage checks, checksums and null candidate
lifecycle fields remain enforced. With `false`, this AP-free pool is rejected.
The function does not rewrite inputs, choose approval identities, record HTTP or
make candidates runnable. Synthetic `fixture-` names cannot bypass candidate
composition checks.

Stored JSON is JCS plus one LF. The set checksum excludes that LF and the root
`set_checksum` member; `review_sheet_sha256` hashes the exact review-sheet bytes.
Recompute both after edits. See also [fixture policy](../../../docs/fixture-policy.md).
The fallback does not waive source completeness or owner review.
