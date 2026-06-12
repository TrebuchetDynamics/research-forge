# Release packaging

Release checklist:

1. Run `go test ./...`, `go vet ./...`, `govulncheck ./...`, and `git diff --check`.
2. Build cross-platform CLI artifacts with `GOOS`/`GOARCH` matrix.
3. Generate checksums: `sha256sum dist/* > dist/checksums.txt`.
4. Generate dependency metadata/SBOM when feasible from `go list -m -json all`.
5. Run install smoke test: `rforge version` and `rforge --help`.
6. Publish release notes from `docs/release-notes-template.md`.

Fyne packaging remains smoke-checked only after the Fyne build decision lands.
