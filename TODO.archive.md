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
