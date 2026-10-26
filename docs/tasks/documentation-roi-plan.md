# Repository-wide documentation ROI plan

Status: Accepted

This is a **standalone planning batch**, not authorization to execute its tasks. Owner request: “high roi improvements in all docs”. Baseline: `721c2466d2b89596abba1d830d9a4f806b6ec660`. Existing code/research approvals, scientific gates, publication limits and the RF-BENCH pause remain unchanged. Planner calls below are derived engineering choices inside accepted workflows, not new product strategy.

## Project model and accepted documentation goal

ResearchForge is a local-first Go research engine used through CLI, passive host integrations/MCP, and a Go + HTMX local UI. Public documentation is also published through the separate Astro/Starlight `site/`. Search folders, project stores, review packages and agent instructions have different artifact/authority boundaries. The [README](../../README.md), [CLI reference](../cli.md), [project format](../project-format.md), [review package contract](../reproducible-review-package.md) and [ADRs](../adr/README.md) own these meanings; implementation/tests remain the source of current executable behavior.

**RF-DOCS (derived engineering goal / planner's call):** make current documentation safe to follow, consistent with implemented commands/artifact paths and straightforward to navigate, while preserving historical evidence and owner decisions. Goals: prevent real-data/fixture confusion, give complete onboarding inputs, distinguish public agent usage from repository-development skills, explain actual setup/check prerequisites, and restore current provenance/navigation links. Non-goals: product changes, visual redesign, bulk rewriting, new generators/check infrastructure, dependency changes, running research, changing host installations, redeploying the site, or requalifying releases/benchmarks. Anti-goals: overwriting real reviews with fixtures, invented inputs, silent install-scope expansion, obsolete blockers presented as current, or historical receipts rewritten as fresh verification.

The most recent report-safety tasks V0-024/V0-025 are already coordinator-closed in current history; the queue has 79 tasks, none open, and nine goals at discovery. Do not duplicate them, alter completed contracts or mark their goals met. No existing task owns the documentation defects below. Use a new RF-DOCS goal without changing existing priorities/focus. The repository runner was idle with zero workers at inspection; do not start/restart/dispatch it. Its separate existing authority may consume committed work independently.

## All-docs inventory and coverage

Read-only `git ls-files` inventory: **287 tracked Markdown/MDX/RST/AsciiDoc files**. This is an inventory-wide pass plus source-backed risk sampling, not a claim to have fact-checked every sentence or external citation. Ignored/generated site output, private jobs and untracked agent artifacts are not documentation owners.

| Area / count at baseline | Role, inspected evidence and disposition |
| --- | --- |
| Root 12 | Front door, glossary, developer guidance, PRD/roadmap and frozen TODO history. Current navigation/setup contradictions feed T3–T5. Preserve PRD/roadmap intent and TODO.archive.md history; add no speculative feature tasks. |
| `docs/` 67 | Current references, decisions, plans, release gates, benchmark contracts and task handoffs. Targeted source-backed read of onboarding, architecture, privacy, owner decisions, package/CLI/release guidance. T4–T6 separate current state from historical criteria and repair retired references. No benchmark-protocol changes. |
| `site/` 7 | All seven current MDX pages read. Workflow/package, public-agent installation and introductory paths feed T1–T3. Sources page has no proved high-impact defect; no forced rewrite. |
| `skills/` 35 | All 35 usage/developer-skill/reference files read by an independent read-only scout. The public skill has a concrete artifact-path error; T2 owns its synchronized copies. Development skills remain development guidance, not the public user pack. |
| `adapters/` 6 + `hermes-plugin/` 1 | Public workflow instructions and mirrored primary skill, not human README files (none present there). Primary mirrors are byte-identical at discovery, including the wrong project filenames. T2 updates all three copies together; five explicit workflow skills otherwise stay intact. |
| `research/` 35 + `benchmarks/` 6 | Historical scientific reports, captured results, protocols and locked evaluation artifacts. Inventory/link scan and targeted cross-check only, not independent scientific review. Preserve bodies/checksums; index their limits in T5. One missing legacy TODO link in `benchmarks/sets/pool-a/review.md:41` is a historical diagnostic, not permission to mutate locked inputs. |
| `opensource/` 25 | Reference inventory/readmes, not newly approved dependencies or verified upstream facts. Inventory/link scan; no proved current defect justifies speculative edits. T5 makes this role discoverable. |
| `examples/` 1 + `internal/` 1 | Example/fixture documentation. Inventory/link scan; retain fixture labels and do not promote examples to live proof. T5 links examples distinctly. |
| `tasks/` 88 | Coordinator-owned goal/task records. Scan only; never worker-edited as ordinary documentation. Preserve IDs, closures, receipts and historical owner direction. |
| `.claude/`, `.pi/`, `.github/` 1 each | Executable release skills or contribution template. Inventory and role classification; no change without a proved instruction-contract issue. Do not treat SKILL.md as harmless prose during planning. |

A fence-aware read-only relative-inline-link scan examined **549 targets** and found **31 candidates**, not 31 independently proved current-doc bugs. Six are outside `tasks/`: five current/handoff references across four documents are handled by T6; the sixth is the historical pool-a reference above. The remaining 25 are task-body source notes with repository-root path conventions; do not mass-rewrite closed task records to satisfy a generic rendered-Markdown checker. Root-absolute site routes, external URLs, reference-style links, arbitrary HTML and external anchor validity are outside this narrow scan's proof. T5 checks authored navigation, not all external literature links.

## Evidence, ranking and chosen approaches

### T1 — safe website review workflow (highest priority)

**OBSERVED, high confidence:** `site/src/content/docs/workflow.mdx:44–49` ends a real-review tutorial with `forge package-fixture`. `internal/forge/forge.go:257–298` requires package-export state and calls `reviewpkg.WriteArtificialPhotosynthesisFixtureProject`, deliberately writing synthetic artifacts into that project. `internal/reviewpkg/fixture.go:383–397` supplies synthetic title/tool/provenance. The same tutorial places `--fetch-pdfs` before screening and runs analysis without the global project prefix. `internal/cli/source_commands.go:1542–1544` performs the requested fetch; the screening-first policy is explicit in [ADR 0008](../adr/0008-screen-on-title-abstract-before-full-text-acquisition.md). These are source observations, not a claim anyone has suffered data loss.

**Planner's call:** make this a conditional real-review workflow map, not an apparently complete copy/paste scientific review. Create/open the project first; supply `queries.txt` explicitly before commands that consume it; search/import metadata, screen title/abstract before OA acquisition, require actual human decisions and accepted evidence before analysis/export. Use `rforge --project ./my-review package create --out ./review.rforgepkg`, followed by audit/replay, only after the documented privacy/artifact prerequisites. `internal/cli/cli.go:1654–1677` and `internal/cli/package_create_test.go` own that normal path. Link deeper canonical docs rather than duplicating every state-machine gate. Remove canned automatic approvals and fixture commands from real-data instructions; any fixture demo is explicitly disposable and separate. Correct project prefixes and prerequisite labels in the cheatsheet/package page too. Rejected alternative: add the missing fixture gates, which would make the wrong operation execute more successfully.

Touches: `site/src/content/docs/workflow.mdx`, `site/src/content/docs/cli.mdx`, `site/src/content/docs/packages.mdx`. No engine changes.

### T2 — correct public agent onboarding and artifact names

**OBSERVED, high confidence:** `site/src/content/docs/agent-skill.mdx:19–30` says Claude/Pi installation but copies every developer skill only into `~/.claude/skills`. `index.mdx:49–58` calls these developer skills the research pack. `install.sh:26–28` deliberately distributes six public skills: research-forge, lit, paper, brief, compare, replicate; `package.json:24–28` supplies Pi registration and Node >=22. All three primary skill copies at line 182 say project creation writes `manifest.json` and `research-forge.lock.json`; `internal/project/project.go:152–153` actually writes `rforge.project.toml` and `rforge.lock.json`. These are separate from a search folder's valid manifest.json.

**Planner's call:** link the canonical [agent installation guide](../agent-install.md) / [installer contract](../install.md); explain six public workflows and passive registration versus development skills. Remove the blanket `skills/*` copy, do not invent another host matrix or reconfigure user homes during validation. On the site index, explicitly use the CLI-only installer profile `bash -s -- --for cli` for its CLI entry and link agent opt-in separately, instead of silently selecting every integration. Builder review also traced index.mdx:40–47 to the query-file reader at source_commands.go:1229–1232/3332–3335: its typical run consumes an unproduced queries.txt. Use inline `--query "artificial photosynthesis"` there instead of `--queries queries.txt`, preserving the project import handoff. This planner-resolvable finding is part of T2, not a new task. Correct only the project-creation sentence in the canonical skill and copy the same bytes to both mirrors. Do not globally replace manifest.json. Preserve scientific human gates, explicit workflows and all other skill behavior. Resources: `agent-docs`.

Touches: `site/src/content/docs/agent-skill.mdx`, `site/src/content/docs/index.mdx`, `skills/research-forge/SKILL.md`, `adapters/skills/research-forge/SKILL.md`, `hermes-plugin/skills/research-forge/SKILL.md`.

### T3 — executable introductory handoff, not invented files

**OBSERVED, high confidence:** `docs/quickstart.md:5–6` runs plain search then imports papers.json, with no producer for that file. `internal/cli/source_commands.go:1160–1185` prints search results, while `internal/cli/library_commands.go:19–42` reads an existing JSON file. The implemented project import route is `executeSearchImport` at source_commands.go:2414–2468. Website getting-started already uses project import, but its unlabeled bare installer defaults to all integrations (`install.sh:90–113`), unlike CLI-only onboarding expectations.

**Planner's call:** use `rforge --project demo search import --source openalex --query "artificial photosynthesis" --pages 1 --limit 20` after project creation in the repository quickstart; retain current valid downstream operations with the project prefix. Mark source calls live, not default offline checks, and make review/report limitations explicit. Make website CLI install use `bash -s -- --for cli`, link optional agent setup and canonical installation details, and ensure every consumed path has a producer or stated user prerequisite. Do not add new fixture data/tests merely for prose or execute public-source examples to validate docs.

Touches: `docs/quickstart.md`, `site/src/content/docs/getting-started.mdx`.

### T4 — current contributor prerequisites and resolved decisions

**OBSERVED, high confidence:** CONTRIBUTING.md:78 and docs/owner-decisions.md:3–17 call licensing an active blocker and prescribe `make decision-issues` / `--issue-body project_license`. The active registry is empty (`internal/cli/cli.go:488–496`); resolved IDs are rejected (`internal/cli/cli_test.go:134`, `TestExecuteDecisionsIssueBodyRejectsResolvedDecisions`). MIT approval is already recorded in docs/owner-decisions.md:32–48 and LICENSE. `docs/1.0-release.md:3` also calls it remaining; this is not permission to assert 1.0 readiness. `docs/developer-setup.md:7` omits Node adapter tests and misnames the audit: Makefile:18–20 runs Go AND `node --test adapters/test`, and check uses `todo-completion-audit`. Root package.json requires Node >=22; go.mod uses Go 1.26/toolchain 1.26.6. Optional Astro site development needs Node >=22.12.0 and npm >=9.6.5 (`site/package-lock.json`, astro engine metadata).

**Planner's call:** present Go/Git/Node prerequisites before make check, distinguish core adapter tests from optional site tooling, and note cold-cache provisioning can need network even when tests are deterministic. Show the actual local gate. Clearly distinguish resolved license history from future generic decision procedure and release revalidation policy. Preserve existing live license gates, required approval fields and audit vocabulary/context; never call them executed or waived here. Stop prescribing the resolved-ID scaffold as a working onboarding command; do not repair the Make target in this docs task. Retain examples for future *active* IDs only when labeled placeholders. No live GitHub calls for doc validation.

Touches: `CONTRIBUTING.md`, `docs/developer-setup.md`, `docs/owner-decisions.md`, `docs/1.0-release.md`.

### T5 — one navigation/currentness map across all docs

**OBSERVED:** there is no tracked docs/README.md. README links selected current guides, but the 287-file inventory mixes current usage, development instructions, accepted plans, historical runs and frozen contracts. CONTEXT.md:13 calls the implemented rforge “planned”; docs/architecture.md omits the now-material shared research/forge/MCP/adapter boundaries. The current implementation packages and the site/astro.config.mjs sidebar support a concise source-linked map, not a new architecture decision.

**Planner's call:** create `docs/README.md` as the human navigation owner, linked from the root README. Group current start/use/build/trust references separately from specs/ADRs and dated research/benchmark/receipt evidence. Cover every inventory area above by category or explicit “historical/reference” pointer, not a hand-maintained 287-row mirror. State which docs are authority for commands, packaging, installation, project artifacts and decisions. Modernize only the glossary's obvious current-tool wording and expand the small architecture map with verified source paths/ownership; preserve stable terminology and accepted ADRs. Include update triggers and the documentation maintenance checks below. No new wiki, route system or generator. Resources: `agent-docs` because README is edited.

Touches: new `docs/README.md`, `README.md`, `CONTEXT.md`, `docs/architecture.md`.

### T6 — repair retired contract links without reviving work

**OBSERVED:** docs/benchmark/spec.md:25 points to deleted TODO.md. Four links in docs/tasks/{rf-bench-offline-round-1,rf-bench-offline-round-2,v0-012-baseline-search-blocker}.md point to removed V0-012-88342f33. Git history records its removal at `192a772`; `git show 3fcac85048bc2c5d744d85f99cf76929562fb4e1:tasks/items/V0-012-88342f33.md` confirms a reachable historical copy. These historical task references are not evidence of a currently runnable task.

**Planner's call:** link benchmark task-contract references to the existing canonical task records named in that section (V-B3, V-B4, V-B4-lock, V-B5, V-B6, V-B6-run, V-D3). For removed V0-012, use a revision-pinned historical GitHub blob at the verified full SHA above and label the link historical/removed from current queue; add a compact currentness note while retaining the old decision text and pause. Do not resurrect the record, change protocol semantics, infer permission from a historic open status, or edit locked pool-a inputs/closed task-body source notes. The 2026-10-23 pause remains fully binding. This is clerical documentation maintenance, not a new benchmark-planning round.

Touches: `docs/benchmark/spec.md`, `docs/tasks/rf-bench-offline-round-1.md`, `docs/tasks/rf-bench-offline-round-2.md`, `docs/tasks/v0-012-baseline-search-blocker.md`.

## Verification and execution setup (proposed, unrun)

Docs-only updates use existing behavior/contract guards and link/example review, not new tests that merely pin prose. Preserve required strings in existing docs tests with accurate historical/revalidation context. Worker-reversible: headings, concise wording, link placement and fixture-free Markdown organization inside each bounded write set. If an owning-path prerequisite blocks checks, repair that scoped doc prerequisite without weakening checks; production logic, manifests and CI remain excluded.

- All tasks: `git diff --check`; `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/docs -count=1`; normal required `make check` at execution/merged-tree gate. Do not call live issue audit targets or tutorial commands during validation.
- T1: existing offline `go test ./internal/cli -run 'TestExecutePackageCreate|TestExecutePackageAuditAndReplay|TestExecuteForgePackageFixture' -count=1`; review package create command/project prefix against its parser; a static search must find no fixture command or --fetch-pdfs in the real-review workflow, no automatic approval boilerplate, and explicit human/precondition gates. Existing fixture tests validate what fixtures are, not the real tutorial's scientific completeness.
- T2: `cmp skills/research-forge/SKILL.md adapters/skills/research-forge/SKILL.md`; `cmp skills/research-forge/SKILL.md hermes-plugin/skills/research-forge/SKILL.md`; `go test ./internal/docs -run TestPassivePluginConformance -count=1`; static manifest/filename/snippet review. Never execute installation snippets.
- T3: `go test ./internal/cli -run 'TestExecuteSearchImportOpenAlex|TestExecuteProjectCreateWritesProject' -count=1`; verify the documented import route with these local fixtures/source, not live queries. Every prerequisite file must be produced or declared.
- T4: `go test ./internal/cli -run 'TestExecuteDecisionsIssueBodyRejectsResolvedDecisions|TestExecuteDecisionsJSONHasNoActiveOwnerBlockers' -count=1`; existing docs tests preserve release-gate fields; source comparison proves gate/runtime wording. No fresh approval claim.
- T5/T6: local Markdown target/anchor review on changed files; root-relative repository paths in coordinator records are not ordinary relative navigation. T6 verifies each replacement task exists and checks the historical blob using isolated Git env; do not fetch or rewrite history.
- Site tasks T1–T3: `npm --prefix site run build`. Execution-only setup, if needed: use locked existing dependencies with `npm --prefix site ci` under Node >=22.12.0/npm >=9.6.5. It may populate node_modules/cache and site/dist but must not change manifests/locks or commit generated output. This is bounded docs-build setup for the later worker, not permission for this planning pass to install anything. No `deploy`, host registration or remote publication. Dependencies are already specified in site/package-lock.json; no new package choice is delegated.

**Verify/direction:** no real-data path ends in a synthetic fixture; all startup inputs and project filenames are correct; public installs are explicitly scoped; current prerequisites/decision states match code; current navigation targets resolve while historical limitations remain visible. No fabricated scalar ROI or “all external facts verified” claim. Guard: existing tests, static source/link review and site build where affected. Stop after the six-task frontier; no experiment, rolling successor, timer or new research daemon.

## Dependencies, resources and decision closure

T1–T6 have no product prerequisites on each other and have disjoint file ownership. T2 and T5 both reserve `agent-docs`, so they serialize despite different files. Other lanes are independently ready subject to actual coordinator ownership/capacity. T3 links existing canonical docs, not uncreated output of T5; no artificial navigation dependency is needed. No task edits docs/cli.md or internal/cli/cli.go, so `cli-surface` is not reserved.

No missing owner intent blocks this batch. New strategy and exhaustive external-citation verification were not requested and are not invented. This document's prescribed execution checks are not claimed as executed by planning.

## Saved batch and builder dispositions

| Priority / outcome | Saved record | Selection/scheduling |
| --- | --- | --- |
| 1 Prevent real-review fixture overwrite and premature acquisition | [V0-026-1a87bee6](../../tasks/items/V0-026-1a87bee6.md) | Now; independent; start by comparing the documented fixture route to the normal package parser. |
| 2 Correct public agent onboarding, mirrored filenames and homepage input | [V0-027-61cd1258](../../tasks/items/V0-027-61cd1258.md) | Now; agent-docs resource. |
| 3 Complete quickstart input/install handoffs | [V0-028-9080850a](../../tasks/items/V0-028-9080850a.md) | Now; independent. |
| 4 Explain real contributor runtimes/checks and resolved license | [V0-029-d1be546d](../../tasks/items/V0-029-d1be546d.md) | Now; independent. |
| 5 Authoritative docs navigation/currentness map | [V0-030-31d485e0](../../tasks/items/V0-030-31d485e0.md) | Now; serializes with V0-027 through agent-docs, not a product dependency. |
| 6 Restore current/historical contract references | [V0-031-026e9f06](../../tasks/items/V0-031-026e9f06.md) | Now; independent clerical lane, no benchmark execution/planning. |

[RF-DOCS](../../tasks/goals/RF-DOCS.md) is unmet, priority 27, focus=false. Existing goals, completed task IDs and owner deferrals remain unchanged. The complete requested planning frontier is the six records above, not a future roadmap count; all documentation areas have a recorded inventory/disposition, including those where no justified edit is proposed.

Independent read-only builder review found one material **planner-resolvable** gap: the homepage's unproduced queries.txt. T2 now explicitly owns an inline-query correction in its scope/acceptance; source_commands.go:3283–3288 confirms the supported inline query path, and saved-record readback checks the resolution. No owner question or new task is needed. **Already answered:** existing docs tests are appropriate without prose-pin tests; locked site setup is explicitly allowed at future execution; historical annotations do not reopen paused work. **Worker-reversible:** headings, wording and link placement. **Owner/external-blocked:** new benchmark campaigns/provider access/scientific qualification remain outside these tasks, not prerequisites. These dispositions are saved in the affected contracts. No known material finding is left unclassified.

Planning checks: capability/prepare API 4 returned READY; artifact/command/role inspection and the bounded read-only scan above were executed. Final affected-record readback, store validation, local link/anchor checks, whitespace validation and normal mandatory commit-hook results are reported with delivery. No site dependencies, public queries, host installation, fixture tutorial or live issue approval check is run during this planning pass. The standalone request stops after committing this plan/task batch.
