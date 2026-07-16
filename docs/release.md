# Release packaging

Release checklist:

1. Run `make license-decision-live-audit` and confirm `make license-decision-approval-gate` passes with `approved:true` before adding or shipping `LICENSE`; if `TODO.md:34` is still unchecked, do not publish a public release. The live decision must include License SPDX identifier, Copyright holder, Approved by, and Approval date.
2. Run the canonical test, audit, and vulnerability gate with `make ci`.
3. Build cross-platform CLI artifacts with `make build-release`.
4. Generate checksums with `make checksums`.
5. Attach `dist/rforge-*` and `dist/checksums.txt` to the GitHub release so `install.sh` can install without Go.
6. Generate dependency metadata/SBOM when feasible from `go list -m -json all`.
7. Run install smoke test: `rforge version` and `rforge --help`.
8. Publish release notes from `docs/release-notes-template.md`.

Run `make web-gui-smoke` to smoke-check the Go + HTMX local web GUI handlers and static workspace before release packaging.
