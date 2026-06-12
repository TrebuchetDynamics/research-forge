# ADR 0004: Use the standard-library CLI parser until command surface stabilizes

## Status

Accepted

## Context

ResearchForge has a broad CLI surface that is still evolving quickly across project, source, library, OSS, parsing, screening, evidence, analysis, and report workflows. The current implementation uses explicit argument parsing in `internal/cli` with integration tests for command behavior and JSON envelopes.

Adopting a framework now would require adapting every command before the command grammar settles. The project also needs local-first tests without pulling extra behavior into normal validation.

## Decision

ResearchForge will continue using a small standard-library CLI parser for now.

A framework may be adopted later if command nesting, completions, generated docs, or plugin-like command registration become difficult to maintain manually. Such a move should include a migration test matrix for existing CLI behavior.

## Consequences

- No new runtime dependency is needed for the current CLI.
- CLI tests remain direct and deterministic.
- Shell completion is implemented as a small static command instead of framework-generated completion.
- Future framework adoption remains possible behind existing command tests.
