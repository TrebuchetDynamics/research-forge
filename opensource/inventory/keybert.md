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

- Citation-linked keyword suggestions from abstracts/passages.
- Reviewer approval workflow before adding suggestions to search plans.
- Keyword diversity/scoring metadata.
- Search-result comparison before/after query expansion.

## Recommended next slice

Add keyword/query-expansion suggestion records with source passage IDs, score, extraction method, reviewer decision, and before/after search-plan provenance.
