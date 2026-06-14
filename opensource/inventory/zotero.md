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

- JSON, CSV, BibTeX, RIS, and CSL-JSON import/export.
- Normalized `PaperRecord` identifiers including DOI, arXiv, PMID, OpenAlex, Crossref, and Semantic Scholar IDs.
- CSL-JSON preserves Zotero-style item IDs in source metadata where available.

Missing features:

- Zotero RDF import/export.
- Better BibTeX citation-key preservation.
- Collection/tag mapping.
- Notes/annotation import.
- Attachment path import with privacy-safe redaction.

## Completed slice

CSL-JSON import/export is implemented. It is simpler and safer than full Zotero RDF, gives immediate Zotero interoperability, and fits the existing library import/export command shape.

Implemented command shape:

```sh
rforge import csl-json zotero-export.json
rforge export csl-json library.csl.json
```

Tests cover DOI normalization, authors, year/date parsing, title/container title, URL, abstract, and round-trip preservation of CSL item IDs where present.

## Recommended next slice

Add Zotero RDF import/export or Better BibTeX citation-key preservation beyond CSL item IDs.
