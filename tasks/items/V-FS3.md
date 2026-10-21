+++
schema = 1
id = "V-FS3"
title = "Publish discoverable, audited scientific research examples"
short = "Publish two genuine research examples"
status = "done"
goal = "RF-BENCH"
section = "Next"
rank = 2000
depends_on = ["V-FS1", "V-FS2"]
touches = ["README.md", "research/frontier-science/README.md", "research/frontier-science/examples.json", "internal/docs/frontier_examples_test.go"]
resources = ["agent-docs"]
kind = "implementation"
ownership = "worker"
scope = "After V-FS1 and V-FS2 finish, curate the latest qualifying real two-topic campaign using the plan's deterministic selection rule. Add README discovery, an examples index and a two-entry path manifest linking the original reports, evidence, source audits, provenance and reproduction instructions. No new research runs, report/receipt rewriting, product fixes, fabricated evidence, extra depth comparison, benchmark changes or push."
acceptance = ["Given absent discovery files, `internal/docs/frontier_examples_test.go::TestFrontierExamplesDiscoverable` first fails. When added, both reports are reachable from root README with correct Majorana/Comprehensive and nickelates/Standard labels and working reproduction links.", "Given incomplete/synthetic campaigns, unsafe paths or missing/tampered artifacts, `internal/docs/frontier_examples_test.go::TestFrontierExamplesRejectIncompleteOrSynthetic` rejects promotion using offline temporary fixtures. Those fixtures never become public scientific examples.", "Given the real selected campaign, `TestFrontierExamplesDiscoverable` checks actual engine run/event identities, source-audit/report/evidence references, nonempty evidence and matching saved output hashes. Mechanical checks never imply scientific or human approval."]
checks = ["`env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/docs -run 'TestFrontierExamples' -count=1`; `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE make check`; run the actual rforge provenance validator offline on both manifest-selected topic indexes; `git diff --check`."]
exclusions = []
closure = "coordinator_verified"
closure_evidence = "tasks/evidence/V-FS3.toml"
+++

- Sources: [public-examples handoff](docs/benchmark/frontier-science-plan.md#frontier-science-stress-test-plan).
- Consumes: `research/frontier-science/<campaign-id>/` with `run.json` (campaign_id, started_at, complete real topic runs), both topic report.md/evidence.csv/provenance.json/citation-audit.json/reproduce.md, actual engine artifacts and separate findings.json; select the latest qualifying started_at, then campaign_id.
- Produces: `research/frontier-science/examples.json` (schema_version=1, ordered majorana/nickelates entries with depth and repository-relative campaign/report/evidence/provenance/citation_audit/reproduce paths); public README links without copying research artifacts.
- Resources: agent-docs
- Ownership: Curation agent, sole writer of the discovery files; source/report corrections remain with V-FS2 before its closeout.
- Live checks: Substantive citation support and appropriate scientific restraint require the attributed source audit, not metadata tests. Preserve source-access limits and unfavorable findings; obey applicable privacy/claim/export gates. Missing scientific support stops promotion, not an excuse for an invented approval.
- Next action: Write `TestFrontierExamplesDiscoverable` failing on missing discovery files, inspect the completed real V-FS2 campaign, then add the smallest linked index/manifest and offline guards. Do not fabricate a campaign if the prerequisite is unmet.
