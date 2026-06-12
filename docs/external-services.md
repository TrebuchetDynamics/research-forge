# External services

ResearchForge keeps heavyweight services optional.

- `rforge service check grobid` reads `RFORGE_GROBID_URL`.
- `rforge service check opensearch` reads `RFORGE_OPENSEARCH_URL`.
- `rforge service check qdrant` reads `RFORGE_QDRANT_URL`.
- `rforge service check r-metafor` reads `RFORGE_RSCRIPT_PATH`.

`service start` and `service stop` are deferred until local runtime ownership is designed.
