# JabRef study note

- Repository/ecosystem: `JabRef/jabref`.
- Area: BibTeX/BibLaTeX library management, citation keys, groups, cleanup, LaTeX workflows.
- Disposition: `pattern-reference`.
- License/action constraint: study BibTeX/BibLaTeX workflows and UX; do not port Java code, icons, localization, or bundled assets without review.

## Why it matters

JabRef is a strong reference-manager model for LaTeX-heavy fields such as physics, mathematics, engineering, and computer science. ResearchForge already imports/exports BibTeX, but JabRef highlights deeper quality control around citation keys, groups, linked files, cleanup, and field normalization.

## Patterns to learn

- Citation keys are user-facing stable identifiers, not disposable metadata.
- BibTeX/BibLaTeX fields require reviewable cleanup because automated normalization can break manuscripts.
- Groups/saved searches are useful for systematic-review subsets and research maps.
- Linked files need privacy and path-redaction handling before export.

## ResearchForge status

Implemented nearby capabilities:

- BibTeX import/export with golden tests.
- CSL-JSON and Zotero RDF metadata paths.
- Duplicate report/merge/split UX.
- Project archive redaction tests for local paths.

Missing features:

- BibLaTeX-specific field preservation beyond current BibTeX coverage.
- Collision repair workflow for reviewer-selected citation-key changes.
- JabRef-style groups/saved searches mapped into ResearchForge collections.
- BibTeX cleanup diff UI before applying normalization.

Implemented:

- `BuildJabRefQualityReport` and `rforge library jabref-quality` report citation-key collisions/missing keys, groups/saved searches, field cleanup diffs, linked-file privacy context, and reviewer-approved normalization without mutating records.

## Recommended next slice

Add reviewer-driven citation-key repair and BibTeX cleanup application flows on top of the existing quality report, preserving before/after diffs and approval provenance.
