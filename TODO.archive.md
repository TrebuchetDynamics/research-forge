# Completed tasks

<!-- archived from TODO.md -->

Source section: 12. Verifiable research engine cycle

<a id="v0-1"></a>

- [x] V0-1 — Land the in-progress workflow commands
- Status: done
- Goal: RF-V0 — Land and harden the in-progress research commands.
- Evidence: done 2026-10-19 in `d8a3fda`; staged-tree `go build`, `go vet`, `gofmt -l`, full `go test ./...` and the pre-commit gate passed. Deviation: `README.md` was included because every hunk documents these commands; OSS-inventory edits and the deleted `.claude/skills/rforge-release/` stay uncommitted.

<!-- archived from TODO.md -->

Source section: Now

<a id="v0-4"></a>

- [x] V0-4 — Isolate git subprocesses from inherited hook environment
- Status: done
- Goal: RF-V0 — Land and harden the in-progress research commands.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan); incident 2026-10-19 06:27 (first runner session): the pre-commit hook ran `make check` from the session worktree `.worktrees/pi-autowork-20261019-062700`; Git exported absolute `GIT_DIR`/`GIT_WORK_TREE`/`GIT_INDEX_FILE`, so the fixture helpers in `internal/oss/clone_test.go` and `internal/cli/cli_test.go` ran `git init`/`config`/`add`/`commit` against this repository. Result: `core.bare=true` (runner crash loop "Not inside a Git repository"), test identity `ResearchForge Test <test@example.org>` in `.git/config`, and commit `80ea31c "initial"` (README replaced by `# fake repo`) on the session branch. Repaired by hand: `core.bare=false`, test identity removed, session branch mixed-reset to `934f928` (stray commit kept in reflog, not pushed).
- Scope: `oss.IsolatedGitEnv` drops repository-locating `GIT_*` variables; `GitCloneRunner` (`rforge oss clone`) and both fixture helpers use it; the pre-commit hook runs `make check` without those variables.
- Acceptance: a regression test pointing `GIT_DIR`/`GIT_WORK_TREE`/`GIT_INDEX_FILE` at a decoy repository leaves it untouched; replaying `go test ./internal/oss ./internal/cli` with that environment leaves the decoy with no commits and an unchanged config.
- Checks: go test ./internal/oss ./internal/cli; make check
- Evidence: fixed in `4b926d7`; red then green on `TestGitCommandsIgnoreInheritedRepositoryEnvironment`; decoy replay of the full `oss` and `cli` suites passed with the decoy unchanged; full pre-commit gate passed.
- Dependencies: None.
- Ownership: ResearchForge maintainer.
- Touches: internal/oss/clone.go, internal/oss/clone_test.go, internal/oss/git_env_test.go, internal/cli/cli_test.go, .githooks/pre-commit

<!-- archived from TODO.md -->

Source section: Needs decision

<a id="v0-5"></a>

- [x] V0-5 — Decide whether the runner may push main to origin
- Status: done
- Evidence: owner decision 2026-10-19: no push for now. `PI_AUTOWORK_PUSH=0` is in `~/.config/pi-autowork/research-forge.env` and in the live runner's environment (`/proc/<pid>/environ`, PID 3061231). Before that, the runner pushed `fb27574` to `origin/main`, including `d5231d6` (the owner's previously uncommitted OSS-inventory edits and the `rforge-release` skill deletion); its handling is the owner's call. Upstream: TrebuchetDynamics/autowork#3 and #4.
- Goal: RF-V0 — Land and harden the in-progress research commands.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan); runner log 2026-10-19 06:27:44 "Pushed main to origin/main" (published `d8a3fda`..`934f928`); `PI_AUTOWORK_PUSH` defaults to `1` in the Autowork toolset (`coordinator.py`), so every runner merge into `main` is pushed.
- Scope: the owner chooses: keep pushing (runner merges are published as they land) or set `PI_AUTOWORK_PUSH=0` for this repository's service so merged `main` stays local until the owner pushes. Planner's recommendation: `PI_AUTOWORK_PUSH=0` until the first runner batch has been reviewed, because this cycle's commits change provenance semantics and public plugin names.
- Acceptance: the chosen setting is recorded here and visible in `autowork status .` or the service environment.
- Checks: systemctl --user show 'pi-autowork@research-forge.service' -p Environment
- Dependencies: None.
- Ownership: ResearchForge owner (publication authority).
- Touches: none

<!-- archived from TODO.md -->

Source section: Needs decision

<a id="v0-6"></a>

- [x] V0-6 — Restart the runner after verifying its repository setup
- Status: done
- Evidence: 2026-10-19 07:10 restart after fast-forwarding the session to `10e8000` (fixed hook and tests) and removing its generated untracked `.gitattributes`; the runner re-registered the merge drivers and merged its `.gitattributes` commit; `AGENTS.md` (`87d9fa3`) answers its full-check question; `core.bare=false` and `user.email` stay the owner's after its commits. Dispatch currently waits on the host resource gate (load from other runners). Upstream: TrebuchetDynamics/autowork#5 and #6.
- Goal: RF-V0 — Land and harden the in-progress research commands.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan); runner state `~/.local/state/pi-autowork/d0648c07b9fe8e30`; `.git/config` merge drivers `docs-sync-goals`/`docs-sync-todo`/`docs-sync-archive` point at release `062d8bc`, which the 06:29 self-update removed; the session worktree holds the runner's `.gitattributes` untracked after the repair; the service restarted every minute against a broken repository with no backoff limit (restart counter above 23).
- Scope: confirm the runner re-registers merge drivers against an existing release on start (or re-run its setup), let it re-commit `.gitattributes` in its session, then `autowork start .` and watch the first dispatch. Report the Autowork-side defects upstream (merge-driver paths not refreshed on self-update; unbounded crash-loop restarts; default push) if the owner wants them filed.
- Acceptance: `autowork status .` shows workers running on Now tasks; `git config core.bare` stays `false` and `git config user.email` stays the owner's after the first worker commit.
- Checks: autowork status .; git config core.bare; git config user.email
- Dependencies: V0-4, V0-5.
- Ownership: ResearchForge owner (runner operation).
- Touches: none

<!-- archived from TODO.md -->

Source section: Now

<a id="v-a1"></a>

- [x] V-A1 — Docs parity check for Go version and skill commands
- Status: done
- Goal: RF-AGENT — Make the agent contract correct: install docs, search depth and provenance levels.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Evidence: Go 1.22+ in the skill vs Go 1.26 in `go.mod`).
- Scope: a Go test run by `make test` reads the `go` directive from `go.mod` and fails when `skills/research-forge/SKILL.md`, `README.md`, `install.sh`, `docs/install.md` or `docs/developer-setup.md` state a different minimum Go version; it also asserts every `rforge <cmd>` named in the skill exists in `topLevelCommands`. Fix the skill's "Go 1.22+".
- Acceptance: the test fails on the current skill text and passes after the fix; introducing a wrong version in any listed file fails it.
- Checks: go test ./internal/docs; make check
- Dependencies: V0-1.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/docs, skills/research-forge/SKILL.md, README.md, install.sh, docs/install.md, docs/developer-setup.md

<!-- archived from TODO.md -->

Source section: Now

<a id="v0-3"></a>

- [x] V0-3 — Fix the provenance event path mismatch
- Status: done
- Goal: RF-V0 — Land and harden the in-progress research commands.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Evidence: provenance path mismatch).
- Scope: writers use `provenance/events.jsonl` while `internal/reviewpkg/package.go` (around lines 117 and 151), `internal/webui/builders.go` (around line 137) and `internal/protocol/live_smoke_snapshot.go` (around line 147) read `data/provenance.jsonl`. Make the writer path canonical through one exported constant in `internal/provenance`, migrate readers, and keep reading the legacy path as a fallback.
- Acceptance: a test appends a project event and sees it in the review package and the web UI view model; a project holding only the legacy file still loads.
- Checks: go test ./internal/provenance ./internal/reviewpkg ./internal/webui ./internal/protocol; make check
- Dependencies: None.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/provenance, internal/reviewpkg, internal/webui/builders.go, internal/protocol/live_smoke_snapshot.go

<!-- archived from TODO.md -->

Source section: Now

<a id="v0-2"></a>

- [x] V0-2 — Harden the new workflow commands
- Status: done
- Goal: RF-V0 — Land and harden the in-progress research commands.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Evidence: in-progress command findings).
- Scope: `rank --out` writes the full ranked list and never overwrites an existing `provenance.json` (write rank provenance as a validator-passing record or a provenance event; surface write errors instead of discarding them); the Unpaywall contact comes from config/env instead of the hard-coded `rforge-resolver@local`; drop "deterministic" from `replicate`/`lit` help unless timestamps are injectable and outputs reproducible; the new commands record `recordCLICommand` events when run with `--project`; add CLI tests for `rank`, `draft` and `paper` and a real offline `replicate` test using an `httptest` server.
- Acceptance: a rank run into a directory that already holds `provenance.json` leaves it byte-identical and writes all ranked records to `--out`; `rforge provenance validate` passes on any provenance file `rank` writes; the resolver sends the configured contact; new tests fail before the fix and pass after.
- Checks: go test ./internal/cli ./internal/sources; make check
- Dependencies: V0-1.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/cli/rank_command.go, internal/cli/draft_command.go, internal/cli/paper_command.go, internal/cli/lit_command.go, internal/cli/history_replicate.go, internal/sources/paper_resolver.go, internal/cli/rank_draft_paper_test.go, internal/cli/history_test.go, internal/sources/paper_resolver_test.go, docs/cli.md

## RF-V0 aggregate proof

- Goal: RF-V0; all six owned tasks V0-1 through V0-6 are already indexed as done. This receipt reconciles the remaining partial goal status; it does not change their acceptance or reopen completed work.
- Sources: [accepted plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan), owner decision 4 and the provenance/in-progress-command findings; original completed task bodies above.
- Implementation evidence: V0-2 worker report `.git/autowork-execution/slot-data/V0-2/attempt-1/report.json` records offline workflow regression and resolver tests passing at `7eaa47ba354f8d3f5815d45dc7fa1518e5817fdc`; V0-3 report at `.git/autowork-execution/slot-data/V0-3/attempt-1/report.json` records canonical/legacy event-path and export regressions passing at `f67dade4b850972e13f513d6725e3bd5fb366517`. Both commits are ancestors of the checked revision. These Git-common-directory reports are local operational evidence, not portable repository links.
- Executed aggregate check: `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE make check`, exit 0, at `e4787cfa703191db2ed892c59b63aa79f2aeab78`, using Go 1.26.6 on linux/amd64. Formatting, module tidy diff, all 30 tested Go packages, vet, TODO completion audit, inventory check (23 entries) and whitespace check passed; the CLI entry package has no tests. Normal tests stayed offline; live opt-in qualifications are not claimed.
- Retained history: V0-1 landing, V0-4 Git-environment isolation, V0-5 no-push decision and V0-6 runner recovery retain their original evidence above. This pass does not rerun owner-only operations, authorize publication, or qualify RF-AGENT, RF-BENCH, RF-MCP, RF-CLAIMS, RF-PACK or RF-RELEASE.

<!-- archived from TODO.md -->

Source section: Now

<a id="v-b2"></a>

- [x] V-B2 — Claim-evidence schema v0 design
- Status: done
- Goal: RF-BENCH — Measure research quality against held-out ground truth.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan); [ADR 0009](docs/adr/0009-verification-facets-and-provenance-levels.md).
- Scope: `docs/claim-evidence-schema.md` plus Go types (no storage yet) in a new `internal/claims` package: claim record, evidence references (identifier, passage ID, content hash, offsets with parser version), five verification facets (source identity, content integrity, passage match, semantic support, human approval) each with state, checker identity and timestamp. Map existing `CitationLockedSuggestion`, `report.ClaimTraceView` and knowledge-graph `report_claim` nodes onto it. Benchmark citation scoring uses these types so later storage (V-D2) is additive.
- Acceptance: JSON round-trip tests for the types; semantic support cannot be constructed with an engine checker; no facet or combination is named or documented as "verified claim".
- Checks: go test ./internal/claims; make check
- Dependencies: None.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/claims, docs/claim-evidence-schema.md

<!-- archived from TODO.md -->

Source section: Now

<a id="v-b1"></a>

- [x] V-B1 — Research-quality benchmark specification
- Status: done
- Goal: RF-BENCH — Measure research quality against held-out ground truth.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Planner's calls: two benchmark tiers; Owner answers: gold sets and agent tier).
- Scope: write `docs/benchmark/spec.md`: question-set schema (question, domain, gold identifiers, gold source review identifier and search date, split `dev`/`held_out`, set checksum); leakage controls (reference review and its identifiers excluded from results and citation-expansion seeds, searches date-capped at the review's search date, question wording never quotes the review title, agents never see the review); HTTP snapshot format for offline replay; metrics (recall@K and overall recall vs gold, precision@K, dedupe true/false merges, requests, wall-clock latency, citation correctness once V-D3 lands); report JSON schema; agent-tier protocol (same model, budgets and questions per pair, N repetitions, paired differences with 95% CIs over questions, blind scoring).
- Acceptance: a builder can implement V-B3 and V-B4 from the spec without further decisions; the term "Research-quality benchmark" from CONTEXT.md is used.
- Checks: make check
- Dependencies: None.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: docs/benchmark/spec.md

<!-- archived from TODO.md -->

Source section: Next

<a id="v-d1"></a>

- [x] V-D1 — Content integrity and stable passage identity
- Status: done
- Goal: RF-CLAIMS — Verify claims against stored evidence with separate facets.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan); [ADR 0009](docs/adr/0009-verification-facets-and-provenance-levels.md).
- Scope: SHA-256 of normalized parsed text per parsed document; a content hash per passage; passage identity stays stable across re-parse when content is unchanged.
- Acceptance: re-parsing unchanged content keeps passage IDs and hashes; a one-character change alters only the affected passage hash.
- Checks: go test ./internal/parsing ./internal/documents; make check
- Dependencies: V-B2.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/parsing, internal/documents

<!-- archived from TODO.md -->

Source section: Next

<a id="v-d2"></a>

- [x] V-D2 — Claim store and CLI
- Status: done
- Goal: RF-CLAIMS — Verify claims against stored evidence with separate facets.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan).
- Scope: append-only `data/claims.jsonl` (via `filetxn`) with `rforge claim add|list|show`; semantic-support entries require agent/model identity and are always `agent_reported`; human approval requires a named reviewer.
- Acceptance: tests refuse engine-attributed semantic support and approval without a reviewer.
- Checks: go test ./internal/claims ./internal/cli; make check
- Dependencies: V-B2, V-D1.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/claims, internal/cli
- Resources: cli-surface — shared `internal/cli/cli.go` command table/help and `docs/cli.md`; tasks holding it run one at a time (2026-10-19: parallel edits there held V-A2 and V-E0)

<!-- archived from TODO.md -->

Source section: Now

<a id="v0-7"></a>

- [x] V0-7 — Redact local paths in the packaged provenance ledger
- Status: done
- Goal: RF-V0 — Land and harden the in-progress research commands.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan); regression from V0-3 (`f67dade`): packages began copying the real `provenance/events.jsonl`, whose events record absolute project paths, so `package audit` failed `content_privacy` for any project under a home directory. Found 2026-10-19 while hand-integrating V-A2: `TestExecutePackageAuditAndReplay`, `TestExecutePackageArchiveRestoreCompatibility` and `TestExecuteForgePackageFixtureCompletesSpineToDone` fail on `main` with a home-directory `TMPDIR` and pass with `TMPDIR=/tmp`, which is why the runner's checks stayed green.
- Scope: when packaging, rewrite the copied event ledger: values inside the project become `project/...`, other values flagged by `security.DetectShareabilityLeak` become placeholders, and the redaction report records the rewrite.
- Acceptance: a regression test with explicit home and Windows paths fails before and passes after, regardless of `TMPDIR`; the three CLI package tests pass with a home-directory `TMPDIR`.
- Checks: go test ./internal/reviewpkg ./internal/cli; make check
- Evidence: fixed in `61b065c`; red then green on `TestCreateRedactsLocalPathsInPackagedProvenance`; the three CLI tests and full `make check` passed with `TMPDIR=/home/xel/.cache/tmp`.
- Dependencies: None.
- Ownership: ResearchForge maintainer.
- Touches: internal/reviewpkg/package.go, internal/reviewpkg/provenance_redaction_test.go

<!-- archived from TODO.md -->

Source section: Now

<a id="v-e0"></a>

- [x] V-E0 — Capability report for compatibility checks
- Status: done
- Evidence: hand-integrated 2026-10-19 as `79606f1` (cherry-pick of `0bd588a`) after the runner held it (TrebuchetDynamics/autowork#7); conflicts in `docs/cli.md` and `internal/cli/cli.go` resolved additively; `go test ./internal/cli ./internal/docs`, `rforge capabilities --json` smoke run and the pre-commit `make check` passed.
- Goal: RF-PACK — Ship passive native plugins for Claude Code, Codex, Pi and Hermes over the same engine.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Workstream E, Versions).
- Scope: add `rforge capabilities --json`: build version, a contract version per command/tool family (`search`, `paper`, `brief`, `compare`, `provenance`, later `mcp`, `claims`) and stable error codes. Compatibility rule used by the skill, plugins and adapters: a release build passes if its version is at least `min_rforge_version` and it reports the required capabilities; a `dev` build passes only on the capability check, never on its version string; a missing or older capability fails with a structured error naming the gap and `install.sh`.
- Acceptance: tests for release-pass, dev-with-capabilities-pass, dev-missing-capability-fail and old-version-fail; `rforge help --json` lists the command.
- Checks: go test ./internal/cli; make check
- Dependencies: None.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/cli/capabilities_command.go, internal/cli/capabilities_test.go, internal/cli/cli.go, docs/cli.md

<!-- archived from TODO.md -->

Source section: Next

<a id="v-a2"></a>

- [x] V-A2 — Search budgets and structured stop reasons
- Status: done
- Evidence: hand-integrated 2026-10-19 as `f71bf8a` (cherry-pick of `7a1f00c`) after the runner held it (TrebuchetDynamics/autowork#7); `docs/cli.md` conflict resolved by keeping both sections; `go test ./internal/search ./internal/sources ./internal/cli ./internal/docs` and the pre-commit `make check` passed (TMPDIR=/tmp; the only failures under a home TMPDIR were the V0-7 regression, which also fails on main without this commit).
- Goal: RF-AGENT — Make the agent contract correct: install docs, search depth and provenance levels.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Owner decision 2; Planner's calls: coverage signal).
- Scope: move batch state and resume out of `internal/cli` into `internal/search`, then add `--depth quick|standard|comprehensive` presets (query-variant and source caps), `--max-requests`, marginal-yield coverage (new unique deduped records per query; saturation after N consecutive low-yield queries, thresholds in the preset) and a structured `stop_reason` (`completed`, `budget_exhausted`, `coverage_saturated`, `interrupted`, `error_threshold`) with requests used and coverage in `manifest.json` and `batch-state.json`. Default depth is standard. Reuse `sources.GraphExpansionBudget` naming and replace the ad-hoc `budget_stopped` raw-ref suffix in `citations expand` with the same stop-reason type.
- Acceptance: offline tests reach each stop reason; resume preserves budget accounting; existing batch/resume tests pass unchanged.
- Checks: go test ./internal/search ./internal/cli ./internal/sources; make check
- Dependencies: V0-1.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/search, internal/cli/source_commands.go, internal/cli/search_batch_state.go, internal/sources/graph_budget.go, docs/cli.md


<!-- archived from TODO.md -->

Source section: Next

<a id="v-a3"></a>

- [x] V-A3 — Skill depth rewrite: Standard by default
- Status: done
- Goal: RF-AGENT — Make the agent contract correct: install docs, search depth and provenance levels.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Owner decision 2).
- Scope: the skill defaults to Standard; escalation to Comprehensive is driven by `stop_reason` and coverage (escalate when a standard run ends `budget_exhausted` without saturation, or the user asks); remove the "Comprehensive default" wording; document stop reasons.
- Acceptance: the V-A1 parity test still passes; no sentence in the skill makes Comprehensive the default.
- Checks: go test ./internal/docs; make check
- Dependencies: V-A2, V0-1.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: skills/research-forge/SKILL.md, SKILLS.md
- Resources: agent-docs — shared `skills/research-forge/SKILL.md`, `README.md` and `docs/agent-install.md`; tasks holding it run one at a time
