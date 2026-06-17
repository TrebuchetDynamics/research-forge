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

- DOAJ metadata connector.
- CORE metadata/full-text candidate connector.
- OA candidate comparison across Unpaywall/DOAJ/CORE.
- License-aware acquisition queue in the HTMX cockpit.

## Recommended next slice

Add a DOAJ/CORE OA discovery adapter that records license metadata, source URL, API provenance, and reviewer approval state before any full-text acquisition.
