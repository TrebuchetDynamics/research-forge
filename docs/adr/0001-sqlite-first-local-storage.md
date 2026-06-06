# ADR 0001: SQLite-first local storage

## Status

Accepted

## Context

ResearchForge is a local-first research workflow engine. The MVP must let a researcher create a project, store scholarly metadata, record provenance, screen records, extract evidence, and generate reproducible reports without operating a database server.

The PRD and development plan mention PostgreSQL for workstation/server deployments and optional SQLite for local single-user mode. Milestone 0 needs a storage foundation before ingestion and screening features can persist domain data.

## Decision

ResearchForge will use SQLite as the default MVP project database.

PostgreSQL remains a future adapter for server/workstation deployments. Storage-facing packages should avoid coupling domain services directly to SQLite-specific details where practical.

## Consequences

- New users can create and inspect local projects without running PostgreSQL.
- Tests can create isolated temporary databases quickly.
- The project format can include a single local database file under each project workspace.
- Migrations must be deterministic and tested because the project database is part of reproducibility.
- PostgreSQL compatibility must be preserved at the application boundary when server mode is introduced later.

## Alternatives considered

### PostgreSQL-first

Pros:

- Strong fit for large deployments and concurrent access.
- Mature indexing, JSON, and operational tooling.

Cons:

- Too much setup for the MVP and local-first user journey.
- Makes early CLI and desktop workflows harder to demo and test.

### SQLite-first

Pros:

- Zero server setup.
- Easy temporary databases in tests.
- Good fit for local project archives and single-user workflows.

Cons:

- Future server/concurrent workflows need a PostgreSQL adapter.
- Some SQL and migration choices need care to avoid SQLite-only assumptions.

## Validation

Initial validation target:

```sh
go test ./...
rforge project create ./tmp/demo --title "Demo Review"
rforge doctor --project ./tmp/demo
```
