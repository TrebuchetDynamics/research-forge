# Threat model

ResearchForge is local-first research software. Primary assets: project manifests, lockfiles, provenance, private notes, restricted PDFs, API keys, and generated reports.

Threats and mitigations:

- Path traversal: project, archive, clone, and document paths must remain inside project-controlled roots.
- External commands: invoke binaries without shell interpolation and store command/version provenance.
- Secrets: redact API keys, emails, local paths, reviewer names, and private notes from shareable outputs.
- Copyright: restricted or local-only documents must not be exported.
- Network APIs: document outbound data flows and keep normal tests on fixtures/mock HTTP servers.
- Archives: extraction must reject absolute paths, parent traversal, and symlinks before restore.
- Retention: project-local data retention policies should be explicit before deletion/export workflows.
