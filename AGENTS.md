# Agent instructions

Full check: make check

- Use red-green-refactor for every change; see [SKILLS.md](SKILLS.md) and [CONTRIBUTING.md](CONTRIBUTING.md).
- Keep normal tests offline and deterministic.
- Git subprocesses must use `oss.IsolatedGitEnv()` (or an equivalent environment without `GIT_DIR`, `GIT_WORK_TREE` and `GIT_INDEX_FILE`), so tests run from a hook in a linked worktree never act on this repository.
