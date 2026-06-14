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
- Evidence support refs linked to passages.

Missing features:

- Rich layered parser-output model beyond parser-run layer counts.
- Rich parser comparison/fallback framework beyond compare reports and run manifests.
- Annotation layer import from Zotero/PDF tools.
- Multiple parser output reconciliation.

## Recommended slice

Extend parser-run manifests into a richer layered document reconciliation workflow.
