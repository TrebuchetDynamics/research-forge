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
