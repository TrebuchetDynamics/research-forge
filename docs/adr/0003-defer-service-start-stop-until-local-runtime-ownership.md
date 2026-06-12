# ADR 0003: Defer service start/stop until local runtime ownership is decided

## Status

Accepted

## Context

ResearchForge has a `rforge service check <name>` command for optional external tools such as GROBID, OpenSearch, Qdrant, and R/metafor. These checks are safe in normal validation because they inspect local configuration and endpoint shape without starting processes, requiring credentials, or depending on live networks.

The TODO includes future `rforge service start <name>` and `rforge service stop <name>` commands where safe/local. Starting and stopping services is a stronger ownership claim than checking configuration. It may involve containers, subprocesses, ports, local data directories, cleanup policy, logs, version pinning, resource limits, and user expectations around processes ResearchForge did or did not create.

ADR 0002 requires normal tests to remain local-first and deterministic, with fake adapters before real external services. That posture should also apply to service lifecycle commands.

## Decision

ResearchForge will defer implementing `rforge service start <name>` and `rforge service stop <name>` until local runtime ownership is explicitly designed.

Before adding start/stop commands, ResearchForge must define:

- which services are safe for ResearchForge to manage locally;
- whether the managed runtime is a subprocess, container, package-managed binary, or user-provided command;
- where runtime state, logs, ports, caches, and data directories live;
- how ResearchForge distinguishes processes it created from user-managed services;
- deterministic fake adapters for normal tests;
- opt-in integration tests for real runtimes.

Until then, `rforge service check <name>` is the supported service command surface.

## Consequences

- Users can inspect optional service configuration without ResearchForge taking process ownership.
- Normal validation stays local-first, deterministic, and free of live runtime dependencies.
- Future start/stop work has a clear design gate instead of growing from ad hoc CLI commands.
- The TODO start/stop items remain open until a service lifecycle module and adapter seam are designed.

## Alternatives considered

### Implement start/stop directly in the CLI

Pros:

- Faster path to demo commands.
- Simple for one local service.

Cons:

- Makes the CLI own runtime lifecycle implementation details.
- Hard to test deterministically without process/container fakes.
- Risks stopping or modifying user-managed services.

### Require users to manage all services externally forever

Pros:

- Small ResearchForge implementation.
- Avoids runtime ownership risk.

Cons:

- Weak local-first user experience for workflows that need optional heavy services.
- Does not leave room for safe managed local adapters later.

### Defer until local runtime ownership is designed

Pros:

- Preserves safe `service check` now.
- Keeps start/stop behind an explicit seam with fake and real adapters.
- Aligns with ADR 0002 local-first validation.

Cons:

- Leaves service lifecycle commands incomplete for now.
- Requires a future design slice before implementation.

## Validation

Current validation remains:

```sh
go test ./...
go vet ./...
git diff --check
```

Future service lifecycle validation should add fake-runtime tests for `service start/stop` before any real local runtime integration tests.
