# Release checklist

Before release:

- `go test ./...` passes;
- race/security checks are handled or explicitly scoped;
- version command reports expected version/commit;
- artifacts have checksums;
- archives exclude `opensource/clones/`, secrets, temp projects, and copyrighted fixtures;
- example project opens with current binary;
- release notes list project-format changes and known limitations.

Publishing to GitHub releases, package managers, or websites requires explicit owner approval.
