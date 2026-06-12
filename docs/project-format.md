# Project format

A ResearchForge project contains:

- `rforge.project.toml` — manifest and project title/schema.
- `rforge.lock.json` — reproducibility lockfile.
- `provenance/events.jsonl` — append-only event log.
- `data/` — local SQLite/JSON stores and generated indexes.
- `documents/` — local-only and open-access document assets.
- `parsed/` — parsed document JSON.
- `opensource/` — OSS notes, scans, reports, and ignored clones.
