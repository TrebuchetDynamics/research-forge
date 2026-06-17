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

- Scientific entity extraction adapter.
- Abbreviation expansion records linked to passages.
- Entity-linking review queue.
- Entity-driven query expansion and extraction schema suggestions.

## Recommended next slice

Add scientific entity suggestion records with mention text, passage ID, offsets, candidate entity IDs, model/version, confidence, and reviewer accepted/rejected/corrected status.
