# Zotero study note

- Repository/ecosystem: `zotero/zotero`, Zotero translators, Better BibTeX ecosystem.
- Area: reference management, citation workflows, collections, tags, notes, attachments.
- Disposition: `pattern-reference`.
- License/action constraint: study UX and interchange formats; do not copy Zotero code or bundled translators without explicit license review.

## Why it matters

Zotero is the dominant open-source reference-manager ecosystem. ResearchForge should interoperate with Zotero libraries rather than asking researchers to abandon existing collections.

## Patterns to learn

- Library-first UX: papers, collections, tags, notes, and attachments are first-class.
- Citation-key preservation matters for LaTeX and manuscript workflows.
- Import/export fidelity matters more than only displaying metadata.
- Attachments and annotations need explicit provenance and path/privacy handling.

## ResearchForge status

Implemented nearby capabilities:

- The forge workflow DAG includes import/dedupe checkpoints with inputs, outputs, provenance actions, and restart-safe skips.
- The local project knowledge graph merges Zotero collections/tags with source concepts, citations, parsed references, evidence, screening, analysis, and report claims for `rforge knowledge query`.
- Citation-to-evidence trace views link report claims back to reference-manager items, accepted evidence, parser passages, source API records, and raw request/response metadata.

- JSON, CSV, BibTeX, RIS, CSL-JSON, and Zotero RDF import/export.
- Normalized `PaperRecord` identifiers including DOI, arXiv, PMID, OpenAlex, Crossref, and Semantic Scholar IDs.
- CSL-JSON preserves Zotero-style item IDs in source metadata where available.
- CSL-JSON imports and exports Better BibTeX `citation-key` values via source metadata.
- CSL-JSON preserves Zotero note/tag metadata and imports attachment filenames with local path privacy redaction.
- Zotero RDF import/export preserves collection metadata through `z:collection` source metadata.

Missing features:

- Zotero RDF import/export.
- Better BibTeX citation-key preservation beyond CSL-JSON metadata.
- Rich collection hierarchy mapping beyond flat CSL/RDF metadata.
- Rich annotation model beyond flat Zotero RDF annotation metadata.
- Attachment import beyond privacy-redacted filenames.

## Completed slice

CSL-JSON import/export and a conservative Zotero RDF import/export subset are implemented. CSL-JSON remains the simplest path for Zotero interoperability, while Zotero RDF supports title, DOI, abstract, venue, date, publisher, Better BibTeX citekey, tags, collections, notes, flat annotations, and privacy-redacted attachment filenames.

Implemented command shape:

```sh
rforge import csl-json zotero-export.json
rforge export csl-json library.csl.json
rforge import zotero-rdf zotero.rdf
rforge export zotero-rdf library.rdf
```

Tests cover DOI normalization, authors, year/date parsing, title/container title, URL, abstract, round-trip preservation of CSL item IDs, Better BibTeX `citation-key` preservation, note/tag/collection/annotation metadata, redacted attachment filenames, and Zotero RDF import/export where present.

## Recommended next slice

Add richer Zotero collection mapping, rich annotation import, and attachment provenance beyond redacted filenames.
