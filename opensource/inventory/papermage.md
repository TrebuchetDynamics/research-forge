# PaperMage study note

- Repository/ecosystem: `allenai/papermage`.
- Area: document object model for scientific papers.
- Disposition: `pattern-reference`.
- License/action constraint: study document-model ideas; do not port code or models without license/provenance review.

## Why it matters

PaperMage models papers as layered entities over text. That pattern can help ResearchForge keep parser outputs, evidence spans, annotations, and citations aligned without tying the core workflow to one parser.

## Patterns to learn

- Use a common document model with typed layers for sections, tokens, references, entities, and annotations.
- Keep provenance for each layer/provider.
- Allow multiple parsers to contribute different layers.
- Make offsets and IDs stable for review/audit workflows.

## ResearchForge status

Implemented nearby capabilities:

- The local project knowledge graph merges PaperMage-style parsed references with collections/tags, OpenAlex concepts, citation edges, evidence, screening, analysis, and report claims for `rforge knowledge query`.
- Citation-to-evidence trace views link report claims back to parser outputs, passage offsets/text, PDFs, accepted evidence, effect-size rows, and source/reference-manager provenance.
- Multi-engine parser arbitration scores GROBID/S2ORC-style/PaperMage/CERMINE/Science Parse/Anystyle outputs per field, routes conflicts to review, records reviewer selection reasons, and emits reconciliation outputs.
- The `/parsing` HTMX arbitration screen compares parser outputs field-by-field with confidence, raw text, offsets, warnings, and accept/correct/reject controls.
- Evidence gap analysis cross-checks the research question, screened-in studies, parsed passages, accepted evidence fields, full-text acquisition, citation-locked claims, and analysis inputs before final inclusion.
- Citation-locked synthesis can draft query expansions, screening rationales, extraction candidates, and report prose only when every suggested sentence has exact source support and remains unaccepted until reviewer review.

- Parsed document model with sections, passages, and references.
- Parser-run manifests record parser name/version, input checksum, parsed output path, layer counts, and warnings.
- GROBID adapter.
- PaperMage-style external JSON adapter via `rforge parse --parser papermage --papermage <file>` for section, paragraph, bibliography, and warning layers.
- Evidence support refs linked to passages.

Missing features:

- Rich layered parser-output model beyond sections/passages/references/warnings.
- Annotation layer import from Zotero/PDF tools.
- Multiple parser output reconciliation.

## Recommended slice

Extend the PaperMage JSON adapter into richer layered document reconciliation with stable offsets, annotation layers, and multi-parser conflict review.
