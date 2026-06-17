# KeyBERT study note

- Repository/ecosystem: `MaartenGr/KeyBERT` and keyword extraction patterns.
- Area: keyword extraction, query expansion, topic labels.
- Disposition: `pattern-reference`.
- License/action constraint: study UX and algorithm pattern; optional adapter only after embedding model/license review.

## Why it matters

ResearchForge can help turn imported abstracts and accepted passages into better source queries and report keywords. Keyword extraction must be citation-linked and reviewer-approved to avoid search bias.

## Patterns to learn

- Keyword suggestions should cite the text they came from.
- Query expansion should preserve the original question and reviewer-approved changes.
- Diversity controls help avoid near-duplicate keywords.
- Model settings and embedding provider determine reproducibility.

## ResearchForge status

Implemented nearby capabilities:

- Search strategy model and query planning scaffolds.
- Retrieval over passages.
- LLM suggestion adapter boundary.
- Provenance event log for user-visible workflow changes.

Missing features:

- Search-result comparison before/after query expansion.
- Optional external KeyBERT adapter after embedding model/license review.

Implemented:

- `DraftQueryExpansionSuggestions` and `ApplyApprovedQueryExpansions` provide KeyBERT-inspired keyword/query-expansion suggestions linked to abstracts/passages/source text, with score and diversity metadata, reviewer approval gates, extraction method provenance, and before/after search-plan provenance when approved terms are applied.

## Recommended next slice

Add search-result comparison before/after query expansion and optional external KeyBERT adapter after model/license review, preserving citation-linked reviewer-approved provenance.
