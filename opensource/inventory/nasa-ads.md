# NASA ADS study note

- Source/API: NASA Astrophysics Data System (ADS).
- Area: physics, astronomy, astrophysics metadata and citation graph.
- Disposition: `adapter-only`.
- License/action constraint: use ADS API under its terms, redact API tokens, honor rate limits, and record requested fields/provenance.

## Why it matters

NASA ADS is a key source for physics and astronomy workflows, which are explicit ResearchForge target domains. It complements arXiv, Crossref, OpenAlex, and Semantic Scholar with bibcodes and astronomy-specific metadata.

## Patterns to learn

- Bibcodes are first-class identifiers in astronomy literature.
- Citation/reference expansion should preserve direction and source fields.
- API tokens must stay outside project manifests and reports.
- Domain-specific metadata needs source-specific normalization rather than lossy generic mapping.

## ResearchForge status

Implemented nearby capabilities:

- Source connector interface and mocked HTTP harness.
- arXiv, Crossref, OpenAlex, Semantic Scholar, PubMed, Europe PMC, and Unpaywall connectors.
- Citation graph expansion commands for supported sources.
- API key redaction tests.

Missing features:

- NASA ADS search connector.
- Bibcode identifier field and normalization rules.
- ADS citation/reference graph expansion.
- Opt-in ADS live smoke test with token redaction.

## Recommended next slice

Add a fake-backed NASA ADS connector for bibcode/DOI/title search that normalizes records into `PaperRecord` while preserving raw ADS IDs and request provenance.
