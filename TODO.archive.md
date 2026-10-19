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
