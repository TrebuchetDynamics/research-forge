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

<!-- archived from TODO.md -->

Source section: Next

<a id="v-a4"></a>

- [x] V-A4 — Provenance levels computed by the validator
- Status: done
- Evidence: hand-integrated 2026-10-19 as `8b6408e` (cherry-pick of `3ae5cbe`) after the runner held it on a merge conflict; `internal/cli/source_commands.go` resolved against V-A2 (both sets of JSON fields, request-accounting preservation then snapshot rollback, single `output.commit()`); `internal/reviewpkg/package.go` merged cleanly with V0-7 redaction, which leaves the run-matching fields intact; `go test ./internal/provenance ./internal/reviewpkg ./internal/cli` and the pre-commit `make check` passed with a home-directory `TMPDIR`.
- Goal: RF-AGENT — Make the agent contract correct: install docs, search depth and provenance levels.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan); [ADR 0009](docs/adr/0009-verification-facets-and-provenance-levels.md).
- Scope: `provenance.json` schema v2 adds `provenance_level`, `run_id` and `outputs` with sha256. Every engine run (`search batch`, `citations expand`, `rank`, `lit`, benchmark) writes an immutable `provenance/runs/<run_id>.json` plus an event in `provenance/events.jsonl` inside its output directory, so project-less research directories work. The skill's topic-level `provenance.json` becomes an agent-maintained index of `run_id`s whose computed level is the lowest among its runs and its own entries. `rforge provenance validate` computes the level: `engine_verified` only if every output checksum recomputes and the `run_id` matches an engine event; a declared `engine_verified` that fails is an error; v1 files read as `unverified` with a warning. Event `actor` gains a kind (`engine`/`agent`/`human`), reusing `automation.ActorAgent`/`ActorHuman`. Reports, packages and the web UI display the computed level. The skill's manual fallback writes `agent_reported`. Docs state that a deliberate forger with file access is out of scope.
- Acceptance: tests reject a hand-typed `engine_verified`, detect a tampered output, downgrade v1 and compute the index level as the minimum.
- Checks: go test ./internal/provenance ./internal/cli ./internal/reviewpkg ./internal/report ./internal/webui; make check
- Dependencies: V0-1, V0-2.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/provenance, internal/automation, internal/cli/provenance_command.go, internal/reviewpkg, internal/report, internal/webui, skills/research-forge/SKILL.md


<!-- archived from TODO.md -->

Source section: Next

- [x] V-A4 — Provenance levels computed by the validator
- Status: done
- Evidence: [completed task record](TODO.archive.md#v-a4).
- Goal: RF-AGENT — Make the agent contract correct: install docs, search depth and provenance levels.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan); [ADR 0009](docs/adr/0009-verification-facets-and-provenance-levels.md).
- Scope: `provenance.json` schema v2 adds `provenance_level`, `run_id` and `outputs` with sha256. Every engine run (`search batch`, `citations expand`, `rank`, `lit`, benchmark) writes an immutable `provenance/runs/<run_id>.json` plus an event in `provenance/events.jsonl` inside its output directory, so project-less research directories work. The skill's topic-level `provenance.json` becomes an agent-maintained index of `run_id`s whose computed level is the lowest among its runs and its own entries. `rforge provenance validate` computes the level: `engine_verified` only if every output checksum recomputes and the `run_id` matches an engine event; a declared `engine_verified` that fails is an error; v1 files read as `unverified` with a warning. Event `actor` gains a kind (`engine`/`agent`/`human`), reusing `automation.ActorAgent`/`ActorHuman`. Reports, packages and the web UI display the computed level. The skill's manual fallback writes `agent_reported`. Docs state that a deliberate forger with file access is out of scope.
- Acceptance: tests reject a hand-typed `engine_verified`, detect a tampered output, downgrade v1 and compute the index level as the minimum.
- Checks: go test ./internal/provenance ./internal/cli ./internal/reviewpkg ./internal/report ./internal/webui; make check
- Dependencies: V0-1, V0-2.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/provenance, internal/automation, internal/cli/provenance_command.go, internal/reviewpkg, internal/report, internal/webui, skills/research-forge/SKILL.md


<!-- archived from TODO.md -->

Source section: Next

- [x] V-A3 — Skill depth rewrite: Standard by default
- Status: done
- Evidence: [completed task record](TODO.archive.md#v-a3).
- Goal: RF-AGENT — Make the agent contract correct: install docs, search depth and provenance levels.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Owner decision 2).
- Scope: the skill defaults to Standard; escalation to Comprehensive is driven by `stop_reason` and coverage (escalate when a standard run ends `budget_exhausted` without saturation, or the user asks); remove the "Comprehensive default" wording; document stop reasons.
- Acceptance: the V-A1 parity test still passes; no sentence in the skill makes Comprehensive the default.
- Checks: go test ./internal/docs; make check
- Dependencies: V-A2, V0-1.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: skills/research-forge/SKILL.md, SKILLS.md
- Resources: agent-docs — shared `skills/research-forge/SKILL.md`, `README.md` and `docs/agent-install.md`; tasks holding it run one at a time


<!-- archived from TODO.md -->

Source section: Next

<a id="v-d3"></a>

- [x] V-D3 — `rforge claim check` deterministic facets
- Status: done
- Goal: RF-CLAIMS — Verify claims against stored evidence with separate facets.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan).
- Scope: source identity (identifier resolves and metadata matches), content integrity (hash recomputes), passage match (exact, then whitespace/Unicode-normalized match of quoted text against the stored passage, with offsets). Wire `CitationLockedSuggestion.ExactText` and `report.BuildCitationEvidenceTraceView` to passage match. Add the citation-correctness metric to the benchmark.
- Acceptance: a quote absent from the stored passage fails passage match while source identity can still pass; facets are reported separately.
- Checks: go test ./internal/claims ./internal/evidence ./internal/report ./internal/benchmarks; make check
- Dependencies: V-D2.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/claims, internal/evidence, internal/report, internal/benchmarks


<!-- archived from TODO.md -->

Source section: Next

<a id="v-b3"></a>

- [x] V-B3 — Engine-tier benchmark harness with offline replay
- Status: done
- Goal: RF-BENCH — Measure research quality against held-out ground truth.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan); `docs/benchmark/spec.md` from V-B1.
- Scope: `rforge benchmark run --set <dir> [--replay|--record]` replays recorded snapshots through the real connectors and dedupe, enforces each set's leakage controls (fail the run if the reference review appears), scores against gold and emits the spec's report JSON with set checksum and rforge version. `--record` is opt-in live. Retire the hardcoded `DefaultCrossToolBenchmarkInput` numbers (remove them or label them `demo`).
- Acceptance: a deterministic replay test on a tiny legal fixture set produces byte-identical reports twice; a leaked reference review fails the run.
- Checks: go test ./internal/benchmarks ./internal/sources ./internal/cli; make check
- Dependencies: V-B1, V-A2.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/benchmarks, internal/sources, internal/cli/cli.go, docs/cli.md
- Resources: cli-surface — shared `internal/cli/cli.go` command table/help and `docs/cli.md`; tasks holding it run one at a time (2026-10-19: parallel edits there held V-A2 and V-E0)


<!-- archived from TODO.md -->

Source section: Next

<a id="v-e1"></a>

- [x] V-E1 — Skill-only plugin scaffolding
- Status: done
- Goal: RF-PACK — Ship passive native plugins for Claude Code, Codex, Pi and Hermes over the same engine.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Workstream E). Reference structure: Autowork's `.claude-plugin/`, `.codex-plugin/`, `.agents/plugins/marketplace.json`, `package.json` with `adapters/pi.mjs`, `plugin.yaml` and `docs/cross-host.md` at https://github.com/TrebuchetDynamics/autowork; never copy its Hermes `plugins/defaults.json` runtime defaults.
- Scope: Claude Code plugin and marketplace listing explicit skill paths; Codex plugin and marketplace; Pi package (`package.json` with `pi.skills`/`pi.extensions`, `adapters/pi.mjs` registering no tools yet); Hermes `plugin.yaml` (`kind: standalone`, no `requires_plugins`, no defaults file; a minimal Python entry point mirroring Autowork's `__init__.py` is allowed if Hermes needs one, with a Python test). Names: plugin and marketplace `research-forge`; Pi commands prefixed `rforge_`. Expose only `skills/research-forge` (verify hosts accept a single-skill path; otherwise expose it via `adapters/skills/research-forge` as a relative symlink or a generated copy checked by the test). Manifest versions agree with each other; docs/release.md gains a step asserting they equal the release tag; the skill frontmatter declares `min_rforge_version` and its required capabilities (rule in V-E0). Write `docs/agent-install.md` with install/upgrade/remove per host; README points to it and keeps the curl path as the generic fallback; curl-install users are told to delete the old skill copy before installing the plugin.
- Acceptance: an offline conformance test (Go, run by `make test`) checks that manifests parse, versions agree, exposed skills are exactly the allowed set (no `*-tdd`), and no manifest contains hooks, agents, settings, permission/approval keys, schedulers, autostart/background entries or install scripts; `adapters/pi.mjs` uses no `child_process` until V-E3.
- Checks: go test ./internal/docs; make check
- Dependencies: V-A1, V-E0.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: .claude-plugin, .codex-plugin, .agents/plugins, package.json, adapters, plugin.yaml, internal/docs, docs/agent-install.md, docs/release.md, README.md, skills/research-forge/SKILL.md
- Resources: agent-docs — shared `skills/research-forge/SKILL.md`, `README.md` and `docs/agent-install.md`; tasks holding it run one at a time


<!-- archived from TODO.md -->

Source section: Next

<a id="v-c1"></a>

- [x] V-C1 — Extract research services from the CLI
- Status: done
- Goal: RF-MCP — Expose the engine through a thin MCP adapter over shared services.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Owner decision 8).
- Scope: move `rank`, `brief`, `compare`, `lit` and `history` logic from `internal/cli` into service packages that return typed results; the CLI keeps flag parsing and rendering only.
- Acceptance: existing CLI tests pass unchanged; new service tests cover each moved function; `internal/cli` no longer holds the moved logic.
- Checks: go test ./...; make check
- Dependencies: V0-2, V-A2.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/research, internal/cli/rank_command.go, internal/cli/brief_command.go, internal/cli/compare_command.go, internal/cli/lit_command.go, internal/cli/history_replicate.go
- Resources: cli-surface — shared `internal/cli/cli.go` command table/help and `docs/cli.md`; tasks holding it run one at a time (2026-10-19: parallel edits there held V-A2 and V-E0)


<!-- archived from TODO.md -->

Source section: Next

<a id="v-b6"></a>

- [x] V-B6 — Agent-tier protocol and scripts
- Status: done
- Goal: RF-BENCH — Measure research quality against held-out ground truth.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Owner answers: agent tier).
- Scope: runbook and scripts under `benchmarks/agent-tier/` per the owner answers: Pi primary, Codex second; same model per pair; equal caps on tokens and search/tool calls (the without-rforge condition uses the agent's native search); blind scoring that strips condition markers (rforge commands, provenance files, run IDs) and randomizes labels before scoring against held-out references with the V-B3 metrics; paired differences with 95% CIs over questions. Verify offline with a stub agent that emits canned outputs. Do not run live agents here (V-B6-run).
- Acceptance: the stub run produces a scored, blinded report; the scorer cannot see condition labels.
- Checks: make check
- Dependencies: V-B1, V-B3, V-A3, V-A4.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: benchmarks/agent-tier, docs/benchmark


<!-- archived from TODO.md -->

Source section: Next

<a id="v-e2"></a>

- [x] V-E2 — Explicit workflow commands
- Status: done
- Evidence: hand-integrated 2026-10-19 as `a30a567` (clean cherry-pick of `0a321ee`) after the runner held it for the protected `package.json` and could not re-admit it once Touches were widened (TrebuchetDynamics/autowork#8); `go test ./internal/docs`, `node --test adapters/test_pi.mjs`, explicit-only checks on all five skills, `rforge_` Pi commands with no subprocess use, and the pre-commit `make check` passed.
- Goal: RF-PACK — Ship passive native plugins for Claude Code, Codex, Pi and Hermes over the same engine.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Workstream E).
- Scope: thin explicit-only skills under `adapters/skills/` for `lit`, `paper`, `brief`, `compare` and `replicate` (`disable-model-invocation: true`; Codex `agents/openai.yaml` with `allow_implicit_invocation: false`); Pi `registerCommand` entries (`rforge_lit` and so on) that load them, Autowork-style. Each states the CLI/MCP call to make and defers methodology to `research-forge`; no research logic in adapters.
- Acceptance: the conformance test covers the new allowed set and fails if any workflow skill is model-invocable.
- Checks: go test ./internal/docs; make check
- Dependencies: V-E1, V0-2.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: adapters, internal/docs, docs/agent-install.md, package.json
- Resources: agent-docs — shared `skills/research-forge/SKILL.md`, `README.md` and `docs/agent-install.md`; tasks holding it run one at a time


<!-- archived from TODO.md -->

Source section: Next

<a id="v-b4"></a>

- [x] V-B4 — Draft candidate gold sets
- Status: done
- Goal: RF-BENCH — Measure research quality against held-out ground truth.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Owner answers: gold sets).
- Scope: draft candidate sets for the pilot, 8 held-out questions plus at least 4 separate dev questions, across at least 3 domains. Each gold set is the included-study list of a published systematic review, recorded with the review identifier and final search date; question wording follows the spec's leakage rules. Save as unlocked candidates under `benchmarks/sets/candidates/` with a review sheet for the owner. Metadata only, per docs/fixture-policy.md. Do not lock or record snapshots. Build on attempt 2 (branch `autowork/V-B4-r2`, commit `5a78a4e`): its `pool-a` (8 held-out, 4 dev, 113 gold records, all machine checks passing) and `discovery.md` artificial-photosynthesis leads. Owner decision 2026-10-19 on artificial photosynthesis (AP): (1) include at least one AP held-out question using a review whose final search is known only to the month, under `search_date_precision: "month"` (spec §2, `b74d761`): cap at the month's first day, cap every search plan there, drop gold published after the cap and list it under "Dropped gold" in `review.md`; the strongest candidate is the 14-paper TiO2 nanotube review (DOI 10.1007/978-3-030-86822-2_10, final search April 2021), whose complete included-paper list must still be verified from public sources; (2) fallback: if no AP review yields a verifiable complete included list, ship the pilot without AP and record in `review.md` and `discovery.md` which AP reviews were checked and why each failed. Never invent dates or list members.
- Acceptance: every candidate validates against the V-B1 schema and `internal/benchmarks` set validation; the review sheet lists each set's source review, date precision, dropped gold and any leakage risk; the pool has at least one AP held-out question or a recorded fallback explaining its absence; the pilot-composition check passes under that rule.
- Checks: make check
- Dependencies: V-B1.
- Ownership: ResearchForge maintainer; Autowork runner may draft; the owner approves sets (V-B4-lock).
- Touches: benchmarks/sets/candidates


<!-- archived from TODO.md -->

Source section: Next

<a id="v-c2"></a>

- [x] V-C2 — Persistent job service
- Status: done
- Goal: RF-MCP — Expose the engine through a thin MCP adapter over shared services.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan).
- Scope: replace the `internal/jobs` stub with persistent jobs (ID, kind, status, progress, `stop_reason`, resume) stored in the research directory and used by search batch; CLI `rforge jobs list|status|resume`. v1 jobs run in-process (CLI or MCP server) and persist after each step, so a job whose host exits is marked `interrupted` and resumes via `jobs resume` from either surface; no detached daemon.
- Acceptance: a killed job is reported `interrupted` and resumes to the same final state as an uninterrupted run.
- Checks: go test ./internal/jobs ./internal/search ./internal/cli; make check
- Dependencies: V-A2.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/jobs, internal/search, internal/cli
- Resources: cli-surface — shared `internal/cli/cli.go` command table/help and `docs/cli.md`; tasks holding it run one at a time (2026-10-19: parallel edits there held V-A2 and V-E0)


<!-- archived from TODO.md -->

Source section: Next

<a id="v-c3"></a>

- [x] V-C3 — `rforge mcp serve` over stdio
- Status: done
- Goal: RF-MCP — Expose the engine through a thin MCP adapter over shared services.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Planner's calls: MCP v1).
- Scope: official Go MCP SDK (`github.com/modelcontextprotocol/go-sdk`). v1 tools: `search` (job-backed, returns a job ID), `job_status`, `source_get` (`sources.ResolvePaper`), `research_status` (brief), `compare`, `evidence_retrieve` (existing retrieve service). No approval, claim-acceptance or export tools. Responses are concise structured JSON with file references, not raw documents. The server does no work until a tool is called.
- Acceptance: a parity test per tool asserts the MCP result equals the CLI `--json` output on the same fixture; for `search`, the completed job's result and manifest equal `search batch --json` on the same snapshot fixture.
- Checks: go test ./internal/mcp ./internal/cli; make check
- Dependencies: V-C1, V-C2.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/mcp, cmd/rforge, internal/cli/cli.go, go.mod, go.sum, docs/cli.md
- Resources: cli-surface — shared `internal/cli/cli.go` command table/help and `docs/cli.md`; tasks holding it run one at a time (2026-10-19: parallel edits there held V-A2 and V-E0)


<!-- archived from TODO.md -->

Source section: Next

<a id="v-c4"></a>

- [x] V-C4 — Standalone MCP documentation
- Status: done
- Goal: RF-MCP — Expose the engine through a thin MCP adapter over shared services.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan).
- Scope: document standalone `rforge mcp serve` setup for any MCP client (plugin-managed registration is V-E3); the skill says MCP and CLI are equivalent and when to use which.
- Acceptance: the documented config starts the server in a smoke test.
- Checks: make check
- Dependencies: V-C3.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: README.md, skills/research-forge/SKILL.md, docs/cli.md
- Resources: cli-surface, agent-docs — shared `internal/cli/cli.go`, `docs/cli.md`, `skills/research-forge/SKILL.md` and `README.md`; tasks holding either run one at a time


<!-- archived from TODO.md -->

Source section: Next

<a id="v-d4"></a>

- [x] V-D4 — Facet-based guards and MCP claim check
- Status: done
- Goal: RF-CLAIMS — Verify claims against stored evidence with separate facets.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan).
- Scope: the report export guard and package audit read facets, with wording that never implies truth; MCP `claim_check` runs deterministic facets and accepts agent-reported semantic support; no approval tool.
- Acceptance: export blocks on a failed passage match; MCP and CLI claim checks agree on a fixture.
- Checks: go test ./internal/report ./internal/reviewpkg ./internal/mcp; make check
- Dependencies: V-D3, V-C3.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/report, internal/reviewpkg, internal/mcp


<!-- archived from TODO.md -->

Source section: Next

<a id="claims-001-82"></a>

- [x] CLAIMS-001-82 — Aggregate offline proof for stored-evidence claim facets
- Status: done
- Goal: RF-CLAIMS
- Payoff: qualify the completed V-D1 through V-D4 work together without treating task closure as goal-level proof.
- Sources: [accepted plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Acceptance: five separate claim facets and facet-based export guards); [ADR 0009](docs/adr/0009-verification-facets-and-provenance-levels.md); [schema](docs/claim-evidence-schema.md#deterministic-checks); [completed V-D1 through V-D4 contracts](TODO.archive.md).
- Evidence: V-D1 (`7a5430a`), V-D2 (`20ab2da`), V-D3 (`43f3faf`) and V-D4 (`cc97226`) worker reports under `.git/autowork-execution/slot-data/<task>/attempt-1/report.json` record passing focused offline checks and no pending live checks. V-D4 was integrated at `68292c5` and closed at `874992b`; RF-CLAIMS still has no goal evidence, and `goals.py validate .` reports `RF-CLAIMS: unmet goal has no open task`. These reports do not themselves record the aggregate full check.
- Scope: record the remaining aggregate offline proof in `docs/tasks/rf-claims-proof.md`, bound to the tested revision, exact commands, exit results and existing test cases covering the original V-D1 through V-D4 contracts. Reuse an existing aggregate result only if its revision and commands can be verified; otherwise run Checks. Do not reimplement completed work, add product scope, run live services or claim scientific truth. Return executed results for normal goal-evidence closeout; no new coordinator receipt or registry is required.
- Acceptance: the recorded proof covers stable unchanged passage IDs/hashes and changed-text hash locality; rejection of engine-attributed semantic support and unnamed human approval; separate facets where a missing quote fails passage match while source identity can pass; facet-based export blocking, package audit and MCP/CLI fixture parity without an approval tool. All listed offline Checks pass on the recorded revision; missing or failed checks remain unqualified, not fabricated passes.
- Checks: `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/parsing ./internal/documents ./internal/claims ./internal/cli ./internal/evidence ./internal/report ./internal/benchmarks ./internal/reviewpkg ./internal/mcp`; `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE make check`.
- Dependencies: V-D1, V-D2, V-D3, V-D4.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: docs/tasks/rf-claims-proof.md
- Next action: verify revision-bound aggregate results, run any missing Checks, and commit the proof document; leave RF-CLAIMS unqualified until executed passing evidence is recorded through `goals.py evidence`.


<!-- archived from TODO.md -->

Source section: Next

<a id="v-e3"></a>

- [x] V-E3 — Typed engine tools per host
- Status: done
- Goal: RF-PACK — Ship passive native plugins for Claude Code, Codex, Pi and Hermes over the same engine.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Workstream E, typed tools per host).
- Scope: Claude Code, Codex and Hermes plugins register `rforge mcp serve` as their MCP server; if Hermes can only register MCP by editing the user's config, the plugin does not edit it and `docs/agent-install.md` gives the line for the user to add. Pi: first verify whether Pi has an MCP client; if not, `adapters/pi.mjs` registers tools mirroring the MCP v1 set that spawn `rforge <cmd> --json` via `execFile` (no shell) with timeouts and return parsed JSON. Every adapter runs the V-E0 compatibility check before its first tool call; a missing or incompatible `rforge` returns a structured error naming `install.sh`. Approval and export operations are not exposed.
- Acceptance: node tests for the Pi adapter with the built binary; the conformance test allows exactly these registrations.
- Checks: go test ./internal/docs; node --test adapters/test; make check
- Dependencies: V-E1, V-E0, V-C3, V-C1.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: .claude-plugin, .codex-plugin, plugin.yaml, adapters, internal/docs, docs/agent-install.md, package.json
- Resources: agent-docs — shared `skills/research-forge/SKILL.md`, `README.md` and `docs/agent-install.md`; tasks holding it run one at a time


<!-- archived from TODO.md -->

Source section: Now

<a id="v-b8"></a>

- [x] V-B8 — Europe PMC as a date-capped recording source
- Status: done
- Goal: RF-BENCH — Measure research quality against held-out ground truth.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan); [benchmark spec](docs/benchmark/spec.md); Owner decision 2026-10-19: the benchmark must not depend on OpenAlex. Pool A's plans used OpenAlex for 24 of 24 calls, the engine qualified only OpenAlex and Crossref, and OpenAlex now meters anonymous use per IP (the first pool A capture got HTTP 429 on 18 of 24 calls).
- Scope: make `europepmc` a qualified benchmark source. The connector (`internal/sources/europepmc.go`) sends only query/pageSize/format: add a publication-date cap applied in the provider query (Europe PMC search syntax, first-publication-date range ending at the cap), cursor paging (`cursorMark`), and raw date metadata (`firstPublicationDate`, electronic and print dates) in `Metadata["publication_dates"]` so record-level date enforcement (`dateReason`) works. In the engine, add the source to the per-call dispatch and to `dateCapped` with the exact cap syntax it accepts. Keep identifiers (PMID, PMCID, DOI) for gold matching. No API key; respect Europe PMC's published rate guidance via the existing HTTP policy.
- Acceptance: offline record-then-replay test through an injected transport (as in `record_test.go`) for a Europe PMC plan: capped query sent, out-of-cap records excluded with reason, replay byte-identical; a Europe PMC plan without the cap fails `date_cap_unenforceable`; spec lists the source and its cap syntax.
- Checks: go test ./internal/sources ./internal/benchmarks; make check
- Dependencies: None.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/sources/europepmc.go, internal/sources/europepmc_test.go, internal/benchmarks/engine.go, internal/benchmarks/validate.go, internal/benchmarks/transport.go, internal/benchmarks/europepmc_record_test.go, docs/benchmark/spec.md
- Resources: benchmark-engine — shared `internal/benchmarks/engine.go`, `internal/benchmarks/validate.go` and `docs/benchmark/spec.md`; tasks holding it run one at a time


<!-- archived from TODO.md -->

Source section: Next

<a id="v-b11"></a>

- [x] V-B11 — Provider-independent live agent comparison
- Status: done
- Goal: RF-BENCH — Measure research quality against held-out ground truth.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan); [benchmark spec](docs/benchmark/spec.md); [agent-tier protocol](benchmarks/agent-tier/README.md); Owner decision 2026-10-19: the benchmark must not depend on OpenAlex. Pool A's plans used OpenAlex for 24 of 24 calls, the engine qualified only OpenAlex and Crossref, and OpenAlex now meters anonymous use per IP (the first pool A capture got HTTP 429 on 18 of 24 calls).
- Scope: before V-B6-run, make the live comparison robust to any single provider being throttled. The with-rforge condition uses a provider mix without OpenAlex by default (`search batch` with `--continue-on-error` records per-provider failures and stop reasons); a throttled provider is skipped and logged, not retried until the run budget is gone. The protocol logs provider failures and rate-limit responses for both conditions and reports runs where either condition lost a provider, so results are never silently biased by an outage.
- Acceptance: a stub-agent run with one provider returning 429 completes, logs the failure in both conditions' records, and the scored report flags the affected runs.
- Checks: make check
- Dependencies: None.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: benchmarks/agent-tier, docs/benchmark


<!-- archived from TODO.md -->

Source section: Next

<a id="v-e4"></a>

- [x] V-E4 — Cross-surface behavior parity suite
- Status: done
- Goal: RF-PACK — Ship passive native plugins for Claude Code, Codex, Pi and Hermes over the same engine.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Owner principles: test behavior).
- Scope: offline CI test across CLI `--json`, MCP and the Pi adapter using the built binary and offline fixture directories: (1) evidence identity, meaning the same record identifiers, dedupe clusters and content hashes; (2) provenance, meaning the same computed level and output checksums; (3) resumability, meaning a job interrupted through one surface and resumed through another reaches the same final state; (4) errors, meaning invalid input, incompatible rforge, budget exhaustion and connector failure yield the same error codes and stop reasons. Comparison normalizes only an explicit allowlist (timestamps, run/job IDs, durations); any other difference fails.
- Acceptance: the suite runs in CI and fails when an injected adapter-only difference is introduced.
- Checks: go test ./internal/mcp; node --test adapters/test; make check
- Dependencies: V-E3, V-C3.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/mcp, adapters/test, Makefile, .github/workflows


<!-- archived from TODO.md -->

Source section: Next

<a id="v-e5"></a>

- [x] V-E5 — Per-host lifecycle qualification runbook
- Status: done
- Goal: RF-PACK — Ship passive native plugins for Claude Code, Codex, Pi and Hermes over the same engine.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan) (Owner requirements: release acceptance).
- Scope: write `docs/qualification/hosts.md` and a receipt template covering, for Claude Code, Codex, Pi and Hermes: install from a local checkout and from the published Git source; upgrade between two versions; remove (host config left clean, approval/permission/schedule settings unchanged before and after, a leftover curl-installed skill detected and reported); recovery (rforge missing, wrong version, MCP server crash, interrupted job resume). Live runs are V-E5-run.
- Acceptance: each host section lists exact commands and the observations a receipt must record.
- Checks: make check
- Dependencies: V-E3.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: docs/qualification


<!-- archived from TODO.md -->

Source section: Next

<a id="v-b12"></a>

- [x] V-B12 — Optional OpenAlex API key, never required
- Status: done
- Goal: RF-BENCH — Measure research quality against held-out ground truth.
- Sources: [plan](docs/verifiable-engine-plan.md#verifiable-research-engine-plan); [benchmark spec](docs/benchmark/spec.md); Owner decision 2026-10-19: the benchmark must not depend on OpenAlex. Pool A's plans used OpenAlex for 24 of 24 calls, the engine qualified only OpenAlex and Crossref, and OpenAlex now meters anonymous use per IP (the first pool A capture got HTTP 429 on 18 of 24 calls).
- Scope: let the OpenAlex connector send an API key from the environment (`RFORGE_OPENALEX_API_KEY`) when set, for users who have one. Nothing in the benchmark, tests or skill may require it. The key must never appear in snapshots, provenance, reports or logs (extend the existing credential redaction and add a test).
- Acceptance: with the key set, requests carry it and recorded snapshots contain no trace of it; with it unset, behavior is unchanged.
- Checks: go test ./internal/sources ./internal/benchmarks; make check
- Dependencies: None.
- Ownership: ResearchForge maintainer; Autowork runner may execute.
- Touches: internal/sources/openalex.go, internal/sources/openalex_test.go, internal/benchmarks/transport.go, internal/benchmarks/openalex_key_test.go
- Resources: benchmark-engine — shared `internal/benchmarks/engine.go`, `internal/benchmarks/validate.go` and `docs/benchmark/spec.md`; tasks holding it run one at a time
