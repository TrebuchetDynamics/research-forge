+++
schema = 1
id = "RF-V0"
title = "Land and harden the in-progress research commands"
priority = 10
source = "docs/verifiable-engine-plan.md#verifiable-research-engine-plan"
status = "met"
focus = false
depends_on = []
+++

- Evidence: executed: 4b926d7: TestGitCommandsIgnoreInheritedRepositoryEnvironment red then green; decoy-env replay of go test ./internal/oss ./internal/cli left decoy unchanged; make check via pre-commit passed (pass)
- Evidence: executed: 61b065c: TestCreateRedactsLocalPathsInPackagedProvenance red then green; package CLI tests and make check pass with home TMPDIR (pass)
- Evidence: executed: env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE make check (revision e4787cfa703191db2ed892c59b63aa79f2aeab78; Go 1.26.6 linux/amd64; TODO.archive.md#rf-v0-aggregate-proof) (pass)
