# Release packaging

Release checklist:

1. Run `make license-decision-live-audit` and confirm `make license-decision-approval-gate` passes with `approved:true` before adding or shipping `LICENSE`; if `CHECKLIST.md:34` is still unchecked, do not publish a public release. The live decision must include License SPDX identifier, Copyright holder, Approved by, and Approval date.
2. Before tagging, set all four plugin manifest versions to the release tag without its `v` prefix: `adapters/.claude-plugin/plugin.json`, `adapters/.codex-plugin/plugin.json`, `package.json` and `plugin.yaml`. Assert equality with `RFORGE_RELEASE_VERSION="$TAG" go test ./internal/docs -run '^TestPassivePluginConformance$' -count=1` (set `TAG` to the intended release tag, such as `v0.1.22`). Refresh the checked skill copy as described in [agent installation](agent-install.md#maintainer-packaging-notes). Run `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s adapters -p test_hermes.py`. Skill-only plugin releases require the offline conformance gate; per-host live lifecycle qualification is separate and must not be inferred from it.
3. Run the canonical test, audit, and vulnerability gate with `make ci`.
4. Build cross-platform CLI artifacts with `make build-release`.
5. Generate checksums with `make checksums`.
6. Attach `dist/rforge-*` and `dist/checksums.txt` to the GitHub release so `install.sh` can install without Go.
7. Generate dependency metadata/SBOM when feasible from `go list -m -json all`.
8. Run install smoke test: `rforge version` and `rforge --help`.
9. Publish release notes from `docs/release-notes-template.md`.

Run `make web-gui-smoke` to smoke-check the Go + HTMX local web GUI handlers and static workspace before release packaging.
