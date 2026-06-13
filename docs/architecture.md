# Architecture overview

ResearchForge keeps CLI and local web GUI behavior behind shared Go services. The CLI remains the reproducible workflow engine; the Go + HTMX web GUI visualizes project state and CLI-generated papers, diagrams, meta-analysis outputs, and report artifacts.

- `internal/project` manages workspaces, manifests, lockfiles, and provenance.
- `internal/library`, `internal/sources`, and `internal/search` manage scholarly metadata.
- `internal/documents`, `internal/parsing`, and `internal/retrieval` manage legal full text and passages.
- `internal/screening`, `internal/evidence`, `internal/analysis`, and `internal/report` support review workflows.
- Optional services use adapter seams and mocked tests first.
