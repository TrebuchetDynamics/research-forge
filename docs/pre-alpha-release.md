# Pre-alpha release plan

Pre-alpha is ready when local CLI workflows pass tests for project creation, metadata search with mocked sources, imports/exports, deduplication, legal PDF handling, parsing/indexing, screening, evidence, analysis, report generation, OSS studies, and provenance.

Validation:

```sh
make check
make install-smoke
make build-release
make checksums
make sbom
```
