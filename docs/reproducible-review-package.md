# Reproducible review package

The Reproducible review package is the first "done" artifact for the Meta-analysis spine. It is a portable ResearchForge package that proves a review can be audited and replayed. Report outputs are included, but the package promise is replayable evidence, provenance, statistics, redaction, and checksums.

## Package goals

A package must allow an independent reviewer to answer:

- What question and protocol drove the review?
- Which sources were searched and with what query/version/cursor state?
- Which records were imported, merged, split, screened, excluded, or included?
- Which full-text assets were legally acquired and which were excluded/redacted?
- Which parser outputs and parser-arbitration decisions created accepted passages/references?
- Which evidence items were accepted and what exact source support backs them?
- Which analysis inputs, model settings, tool versions, outputs, warnings, and checksums produced the statistical result?
- Which report claims are supported, weak, blocked, or reviewer-approved?
- Can the package be restored and replayed without private local state?

## Required package layout

```text
review.rforgepkg/
  manifest.json
  checksums.sha256
  redaction-report.json
  replay.sh
  audit-report.json
  project/
    rforge.project.toml
    rforge.lock.json
    data/
      provenance.jsonl
      forge-state.json
      source-plans/
      connector-capabilities.json
      identity-decisions.jsonl
      screening-audit.jsonl
      extraction-schema.json
      evidence.jsonl
      parser-manifests/
      claim-trace.json
    library/
    documents/            # only shareable/approved assets
    parsed/
    analysis/
    reports/
```

The physical archive format may be `.tar`, `.zip`, or a directory during early development, but the logical layout and manifest semantics must remain stable.

## Required `manifest.json` fields

- `schemaVersion`
- `packageID`
- `createdAt`
- `createdBy`
- `researchForgeVersion`
- `projectTitle`
- `question`
- `metaAnalysisSpineVersion`
- `sourcePlanRefs`
- `lockfileRef`
- `provenanceRef`
- `screeningAuditRef`
- `extractionSchemaRef`
- `acceptedEvidenceRef`
- `analysisArtifactRefs`
- `reportRefs`
- `redactionReportRef`
- `checksumManifestRef`
- `replayCommand`
- `auditCommand`
- `warnings`

## Required contents

| Content | Required evidence |
| --- | --- |
| Project config | `rforge.project.toml`, workflow lockfile, package manifest |
| Source plans | query text, filters, connector capability snapshot, cursor/resume state, API/privacy warnings |
| Source records | normalized records plus raw source refs, not necessarily raw restricted payloads |
| Identity/dedupe | duplicate reports, reversible merge/split decisions, conflict status |
| Legal acquisition | OA/license metadata, source URLs, approval decisions, excluded/restricted asset list |
| Parser outputs | parser manifests, versions, command/config, input/output checksums, selected parsed documents |
| Parser arbitration | field/passages/reference conflict decisions and reviewer corrections |
| Screening | decisions by stage, reviewer attribution, conflict/adjudication log, uncertain queue status, PRISMA counts |
| Evidence | extraction schema, accepted evidence, correction history, exact source support links |
| Analysis | input snapshot, effect-size settings, scripts, engine versions, warnings, outputs, forest/funnel artifacts, checksums |
| Report | Markdown/HTML/LaTeX or scaffold outputs, claim traceability matrix, report audit |
| Redaction | local path, credentials, reviewer notes, restricted assets, embedding/cache redactions |
| Replay/audit | replay script/command, audit report, validation receipts |

## Checksum requirements

- Every included file has a SHA-256 entry in `checksums.sha256`.
- Manifest includes the checksum manifest checksum.
- Analysis outputs include checksums already captured by the analysis module.
- Parser raw/normalized outputs include input and output checksums.
- Package audit verifies all checksums before replay.

## Redaction requirements

The package must exclude or redact:

- secrets and credentials;
- private local absolute paths;
- copyrighted/restricted PDFs not approved for sharing;
- private reviewer notes not marked shareable;
- cache files and temporary local clones;
- remote embedding payload caches unless explicitly approved and shareable.

The redaction report must list what was removed or transformed and why.

## Replay requirements

`rforge package replay <package>` must verify, at minimum:

1. manifest schema version is supported;
2. checksums match;
3. lockfile and package manifest agree on tool/parser/model versions;
4. source plans are present and replayable in dry-run/offline mode;
5. screening audit is internally consistent;
6. accepted evidence links resolve to included passages or approved asset refs;
7. analysis input snapshot matches accepted evidence included in the package;
8. report outputs match claim traceability records or are flagged stale;
9. redaction report is present and no known private-path/credential patterns remain.

Full live-service replay is optional and must be opt-in; normal replay must remain offline.

## Audit failure modes

Package audit must fail with actionable error codes for:

- missing required manifest fields;
- missing referenced files;
- checksum mismatch;
- unsupported package schema version;
- unresolved screening conflicts;
- accepted evidence without source support;
- analysis input/evidence mismatch;
- unsupported report claims;
- missing redaction report;
- detected secrets/private local paths;
- restricted document assets included without approval;
- missing parser manifests for included parsed documents.

## Done criteria

The Reproducible review package format is complete when ResearchForge has:

- package create/audit/replay commands;
- deterministic fixture package tests;
- archive/restore/move tests on a temp directory;
- redaction tests for paths, credentials, reviewer notes, restricted documents, and cache files;
- analysis replay checks against known-result fixtures;
- report claim-trace checks;
- documentation for what is and is not included in a shareable package;
- HTMX package preview that shows manifest, redaction, checksums, warnings, and replay status before export.
