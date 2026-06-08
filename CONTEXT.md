# ResearchForge Context

This glossary captures stable project language for ResearchForge. It is not a feature specification; implementation details belong in the PRD, roadmap, development plan, TODO, or ADRs.

## Glossary

### ResearchForge

The open, reproducible research engine for academic literature discovery, systematic review, evidence extraction, meta-analysis, and auditable reporting.

### `rforge`

The planned command-line tool for ResearchForge workflows.

### Research project

A local ResearchForge workspace containing the research question, source records, documents, screening decisions, extracted evidence, analyses, reports, project manifest, lockfile, and provenance.

### Project manifest

The human-readable project configuration that describes a ResearchForge project's research question, sources, schemas, storage mode, external services, and export settings.

### Workflow lockfile

The machine-written record of tool versions, external-service parameters, parser/model versions, and analysis settings needed to reproduce project outputs.

### Project health report

An actionable summary of Research project invariants such as project manifest, Workflow lockfile, and local storage availability, intended for CLI and future Fyne status displays.

### Provenance

The audit trail that records where research data came from, which actions were taken, which tools and parameters were used, and which source material supports claims.

### Paper record

A normalized scholarly metadata entry for a paper or preprint, preserving identifiers, source-specific metadata, and source provenance.

### Document asset

A local PDF, XML, JATS, HTML, text file, or related full-text artifact with acquisition source, legality/OA status, license metadata where available, checksum, and provenance.

### Parsed document

A structured representation of a document asset, including sections, references, passages, and optionally tables, figures, equations, or other scientific content units.

### Passage

A stable, citable unit of parsed document text that can support retrieval results, evidence extraction, and report audit links.

### Screening decision

An include, exclude, or uncertain judgment made during systematic-review screening, with stage, reviewer, reason where applicable, timestamp, and provenance.

### Evidence item

A structured extracted value or claim that is linked to exact source support such as a passage, table, figure, equation, dataset, or citation.

### Analysis run

A reproducible statistical execution over accepted evidence, including inputs, model settings, scripts or notebooks, tool versions, outputs, warnings, and checksums.

### Report build

A generated research report export with citations, evidence tables, screening summaries, analysis outputs, audit appendix, and build metadata.

### OSS repository study

A ResearchForge record of an open-source project studied for possible integration or design reference, including metadata, license, risks, notes, and integration decisions.

### OSS study disposition

The classification for how ResearchForge may use an OSS repository study: `pattern-reference` by default, or explicitly escalated to `adapter-only`, `integrate`, `needs-license-review`, or `avoid` before any dependency, code, schema, fixture, or asset is used.

### Artificial photosynthesis fixture topic

The main deterministic end-to-end test topic for ResearchForge. Use the spelling "artificial photosynthesis" in code, fixtures, docs, and test names.

### Repository-embedded ResearchForge config

A `.researchforge` configuration file or directory created in the repository where `rforge` is run. It records repo-local ResearchForge settings without requiring users to leave their working repository.

### Default ResearchForge workspace

The default Research project folder for repo-embedded use: `<repo>/research-forge/`. ResearchForge should assume a repository may already contain academic files, PDFs, notes, or other research assets and should discover or import them only through explicit, provenance-recorded workflow steps.

### Retrieval-first, provenance-first, statistics-first, LLM-assisted

The core ResearchForge principle: retrieve and cite source material first, preserve provenance, use auditable statistical methods, and allow LLMs only as assistants for tasks such as query expansion or extraction suggestions.
