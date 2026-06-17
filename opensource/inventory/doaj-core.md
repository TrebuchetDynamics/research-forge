# DOAJ / CORE study note

- Source/API: Directory of Open Access Journals (DOAJ) and CORE.
- Area: open-access discovery, license metadata, full-text candidates.
- Disposition: `adapter-only`.
- License/action constraint: use APIs under attribution/rate-limit rules; CORE credentials and redistribution constraints must be documented; do not treat OA discovery as automatic redistribution permission.

## Why it matters

ResearchForge needs legal full-text discovery beyond Unpaywall, especially for open-access articles and interdisciplinary literature. DOAJ and CORE can help identify OA metadata, journal policy, and full-text candidates.

## Patterns to learn

- License metadata must be captured at acquisition time.
- OA status and full-text URL are not enough; shareability/reuse terms matter.
- API credentials and rate limits need source-specific provenance.
- Candidate downloads should enter a review queue before storage/export.

## ResearchForge status

Implemented nearby capabilities:

- Unpaywall OA lookup and legal PDF URL selection.
- PDF fetch commands with copyright/OA guard tests.
- Document assets with license, OA status, checksum, local path, and MIME type.
- Privacy/copyright documentation and shareable-report redaction tests.

Missing features:

- Richer live-service drift/rate dashboards for DOAJ and CORE.

Implemented:

- DOAJ and CORE connectors normalize open-access metadata and full-text candidates with license metadata, source URL provenance, attribution/rate-limit policy metadata, API provenance, OA candidate comparison across Unpaywall/DOAJ/CORE, license-aware acquisition queues, and reviewer-approved acquisition gates before download/archive use.

## Recommended next slice

Add richer live-service drift/rate dashboards and optional CORE credential budget reporting on top of implemented DOAJ/CORE discovery and acquisition gates.
