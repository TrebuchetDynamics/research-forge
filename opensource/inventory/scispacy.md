# SciSpaCy study note

- Repository/ecosystem: `allenai/scispacy`.
- Area: scientific and biomedical entity recognition, abbreviation detection, entity linking.
- Disposition: `adapter-only`.
- License/action constraint: optional NLP adapter only after model license and biomedical text-handling review; do not vendor models or corpora.

## Why it matters

Entity extraction can support query expansion, evidence schema suggestions, biomedical full-text workflows, and domain maps. ResearchForge needs this as reviewer-audited suggestions, not silent ontology truth.

## Patterns to learn

- Entity mentions need offsets and source passages.
- Entity linking should expose candidate IDs and confidence.
- Abbreviation resolution matters for scientific literature.
- Model/domain mismatch must be visible to reviewers.

## ResearchForge status

Implemented nearby capabilities:

- Parsed passages with stable IDs.
- Evidence suggestions adapter interface.
- PubMed/Europe PMC source connectors and biomedical live smoke docs.
- Retrieval and source-link view models.

Missing features:

- Real optional SciSpaCy adapter after model/license/biomedical text-handling review.
- Entity-driven query expansion and extraction schema suggestions.

Implemented:

- `DraftScientificEntitySuggestions`, `EveryScientificEntitySuggestionAuditable`, and `rforge evidence entity-suggest|entity-review` create SciSpaCy-inspired scientific entity suggestions with passage IDs/offsets, abbreviation resolution, entity-link candidates, confidence, model provenance, and reviewer accept/correct/reject decisions.

## Recommended next slice

Add a real optional SciSpaCy adapter and entity-driven query expansion/extraction-schema suggestion workflows while preserving auditable suggestion queues.
