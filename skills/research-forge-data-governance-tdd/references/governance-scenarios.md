# Governance scenarios

Test these early:

- old project manifest loads and upgrades predictably;
- unknown future manifest version fails safely;
- project archive restore preserves checksums and provenance;
- shareable export redacts local absolute paths and secrets;
- OA policy rejects non-OA PDF download unless manually imported with explicit local-only status;
- lockfile records external tool versions used in parsing/indexing/analysis;
- migration failure leaves a recoverable backup.
