---
name: research-forge-release-packaging-tdd
description: Build ResearchForge release, install, and packaging workflows with tests first. Use for versioning, goreleaser, cross-platform builds, web GUI packaging, checksums, archives, installers, upgrade tests, or release notes.
---

# ResearchForge Release Packaging TDD

Use this skill when preparing repeatable builds or releases.

## Quick start

1. Identify the release behavior or artifact guarantee.
2. Write a failing build/test/script check first.
3. Implement minimal packaging automation.
4. Verify artifacts are reproducible enough and checksummed.
5. Update release docs.

## TDD contract

- **Red:** failing test or CI script for version output, archive contents, checksums, install smoke, or upgrade compatibility.
- **Green:** smallest packaging change.
- **Refactor:** reduce manual release steps and centralize metadata.
- **Receipt:** build/test command output and artifact list.

## Slice areas

- `rforge version` build metadata.
- Cross-platform CLI builds.
- Local web GUI packaging/static-build smoke checks after stack selection.
- Checksums and SBOM/dependency metadata.
- Example project bundled as release fixture.
- Upgrade tests for project format.
- Release notes generation.
- Install documentation.

## Verification gate

Done requires:

- release artifact behavior is checked by tests/scripts;
- version/commit/date metadata is visible;
- checksums are generated for artifacts;
- install or smoke command is documented.

## Red lines

- Do not publish a release without explicit user approval.
- Do not include test secrets, local paths, clones, or copyrighted assets in artifacts.
- Do not break old project formats without migration tests and release notes.

## References

- [Release checklist](references/release-checklist.md)
