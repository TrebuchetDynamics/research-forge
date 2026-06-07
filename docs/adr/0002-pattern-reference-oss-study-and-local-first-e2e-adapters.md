# ADR 0002: Pattern-reference OSS study and local-first e2e adapters

## Status

Accepted

## Context

ResearchForge is intended to learn from mature open-source scholarly tools while remaining product-owned, legally clean, local-first, and reproducible. The PRD names projects such as GROBID, Zotero, JabRef, ASReview, OpenSearch, Qdrant, and R `metafor` as systems to study or integrate with.

The project also needs broad end-to-end validation across workflows such as project creation, scholarly search, parsing, screening, evidence extraction, analysis, and report builds. Normal tests must remain deterministic and must not depend on live networks, credentials, heavyweight services, private data, copyrighted PDFs, or third-party source code copied into this repository.

## Decision

ResearchForge will classify every OSS repository study as `pattern-reference` by default.

A study may be escalated only through an explicit disposition:

- `pattern-reference` — learn workflow, interface, testing, and architecture ideas without copying code, schemas, fixtures, docs, UI assets, or other project artifacts.
- `adapter-only` — call the external tool through an adapter such as HTTP, CLI, subprocess, or container.
- `integrate` — add the project as a dependency only after license, security, maintenance, and fit review.
- `needs-license-review` — pause implementation use until human license review resolves the risk.
- `avoid` — do not use because of license, maintenance, security, scope, or architecture concerns.

ResearchForge will use local-first e2e scenario suites as the primary proof of workflow success. Normal e2e tests must use local fake adapters before real adapters are added to those workflows. Real external services are reserved for opt-in integration tests.

Workflow code must depend on adapter seams rather than live services. Local fake adapters and real adapters should exercise the same workflow interface where practical.

## Consequences

- ResearchForge can study OSS projects widely without contaminating product source or tests.
- Broad e2e tests can cover realistic workflows without network, credentials, live services, or flaky external state.
- Adapter seams must be justified by at least two adapters: local fake and real/opt-in external.
- Real-service coverage still exists, but it is opt-in and separate from normal validation.
- Some early work may take longer because local fake adapters and fixtures are required before real integrations enter workflow tests.

## Alternatives considered

### Copy/adapt code or schemas directly from useful OSS projects

Pros:

- Faster initial implementation for known workflows.
- Might match mature tools more closely.

Cons:

- High license and provenance risk.
- Hard to audit ownership of copied code, schemas, fixtures, or documentation.
- Conflicts with the PRD requirement to avoid copying external source into ResearchForge without human approval.

### Use live external services in normal e2e tests

Pros:

- Validates real integration paths early.
- May catch deployment and service-compatibility issues.

Cons:

- Slow, flaky, credential-dependent, and not reproducible offline.
- Violates the local-first testing posture.
- Makes normal validation dependent on network and third-party availability.

### Unit-test most modules and keep e2e tests narrow

Pros:

- Faster small tests.
- Easier isolated failures.

Cons:

- Misses ResearchForge's user-facing workflow risks.
- Encourages shallow modules and mock-heavy tests that do not prove reproducibility, provenance, or report outputs.

## Validation

Normal validation should include:

```sh
go test ./...
go vet ./...
git diff --check
```

As workflow features are added, default e2e suites should use local fixtures, temporary SQLite projects, fake HTTP servers, fake parser/search/vector/statistics adapters, and golden report outputs. Real GROBID, OpenSearch, Qdrant, R/metafor, GitHub, and scholarly API tests should be opt-in through explicit environment configuration.
