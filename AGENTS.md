# Agent instructions

Full check: make check

- Use red-green-refactor for every change; see [SKILLS.md](SKILLS.md) and [CONTRIBUTING.md](CONTRIBUTING.md).
- Keep normal tests offline and deterministic.
- Git subprocesses must use `oss.IsolatedGitEnv()` (or an equivalent environment without `GIT_DIR`, `GIT_WORK_TREE` and `GIT_INDEX_FILE`), so tests run from a hook in a linked worktree never act on this repository.
- Shared hot files: a task that edits `internal/cli/cli.go` (command table/help) or `docs/cli.md` declares `Resources: cli-surface`; a task that edits `skills/research-forge/SKILL.md`, `README.md` or `docs/agent-install.md` declares `Resources: agent-docs`. Tasks sharing a resource run one at a time; parallel edits to these files cause merge conflicts the runner cannot resolve.
