# BERTopic study note

- Repository/ecosystem: `MaartenGr/BERTopic`.
- Area: topic modeling, clustering, domain maps, research landscape exploration.
- Disposition: `pattern-reference`.
- License/action constraint: study topic-modeling workflow; optional external adapter only after dependency, model, and license review.

## Why it matters

ResearchForge should help researchers understand a domain, not just collect papers. BERTopic-style workflows point toward interpretable topic clusters over abstracts/passages, tied back to exact source papers and queries.

## Patterns to learn

- Topic labels are suggestions and must link to representative documents/passages.
- Model settings and embedding model versions determine reproducibility.
- Topic reduction/merging needs reviewer-visible history.
- Domain maps should integrate with citation graphs and screening status.

## ResearchForge status

Implemented nearby capabilities:

- Citation graph export and web SVG preview.
- OpenAlex concepts/source metadata.
- Retrieval indexes over passages.
- OSS/topic scan metadata workflow for repositories.

Missing features:

- Topic-cluster artifacts over paper abstracts/passages.
- Representative-paper and representative-passage links for each topic.
- Topic merge/split review history.
- Integration with research map dashboard filters.

## Recommended next slice

Add a topic-cluster/domain-map artifact that stores input record IDs, model settings, representative terms/passages, reviewer-edited labels, and links to citation graph neighborhoods.
