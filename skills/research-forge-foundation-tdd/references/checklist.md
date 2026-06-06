# Foundation checklist

Evidence to inspect before each slice:

- `DEVELOPMENT_PLAN.md` Milestone 0
- `RESEARCH-FORGE-PRD.md` sections 5, 6, 7.9, 7.10, 8, 9
- current `go.mod`, `cmd/rforge`, `internal/project`, `internal/provenance`, and CI files when present

Recommended test types:

- package tests for manifest, lockfile, provenance, and storage behavior;
- CLI tests for command output and exit codes;
- golden tests for JSON output when output shape stabilizes.

Initial acceptance demo:

```sh
go test ./...
go run ./cmd/rforge --help
go run ./cmd/rforge project create ./tmp/demo --title "Demo Review"
go run ./cmd/rforge project inspect ./tmp/demo --json
go run ./cmd/rforge doctor --json
```
