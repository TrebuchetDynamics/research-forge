# Hermes installer reliability and local qualification

Status: Accepted

Scope source: owner request, “improve installer for hermes and install in this pc”, through standalone Plan For Me. This occurrence saves the complete plan and tasks, then stops; it does not implement the changes, install software, mutate a Hermes profile or dispatch a worker. The two requested outcomes are installer improvements and subsequent qualification on this PC. Routine choices below are planner's calls, not additional owner decisions.

## Goal, success and boundaries

**Size:** small project, two tasks with a real prerequisite: improve/test the installer offline; then use that landed installer on this PC. Installation depends on the fix, not merely on shared files. There are no independent implementation lanes within this two-outcome slice; unrelated repository work remains independent.

**Goal:** make the Hermes component safe and repeatable, then confirm ResearchForge is usable in the selected local Hermes profile without disturbing existing user state.

**Success:** a tested installer preserves standalone skills and profile isolation, handles an already installed/pinned plugin honestly, and verifies the exact plugin is enabled; a separate local receipt records the real installation/no-op and runtime skill discovery.

**Non-goals:** install or upgrade Hermes itself; redesign plugin packaging; register MCP or edit its configuration; upgrade the existing rforge binary when its capabilities already pass; update Pi/Codex/Claude Code; recreate profiles; deploy to Docker; roll out to every profile again; introduce new source/ref flags, dependencies or a general installer framework.

**Anti-goals:** moving a customized skill in the wrong profile, silently replacing an immutable pin with main, matching a similarly named or disabled plugin as success, suppressing native command failures, bypassing plugin scanning, claiming installation from a dry-run, or claiming runtime discovery from a CLI that omits plugin skills.

**Invariants:** the plugin remains passive; no settings for approvals, credentials, models, memory, hooks, gateways or schedules change. Normal tests stay offline and deterministic. Git subprocesses use `oss.IsolatedGitEnv()` or equivalent scrubbed repository selectors. Preserve unrelated dirty files, task histories and the frozen archive.

## Inspected evidence

Repository HEAD at inspection: `c655c98` (including the working tree, which had only pre-existing untracked agent/runtime artifacts).

- [install.sh](../install.sh), `install_hermes`, always runs native `plugins install <repo>/tree/main/hermes-plugin --no-deps` followed by enable, even when the plugin already exists. `hermes_installed` greps rich/text listing for a substring and does not establish exact identity or enabled status. The update path delegates to native update, which can refuse pinned installs; preserve that refusal rather than invent success.
- `hermes_skills_dir` hardcodes `~/.hermes/skills` for current and `~/.hermes/profiles/<name>/skills` for explicit names, then calls the generic six-skill quarantine. It does not follow `HERMES_PROFILE`, sticky selection or `HERMES_HOME`; explicit `default` is also not a named profile directory. Hermes registers only the methodology skill, so moving all six standalone workflow skill directories is not justified.
- [Existing installer tests](../internal/installer/install_test.go) use isolated HOME and fake CLIs, cover default/explicit/unknown profiles, but the Hermes fake emits only a bare name/status line. Extend that fake only as required to reflect native plain listing; place the new behavioral tests in a focused file.
- Installed upstream source at `~/.hermes/hermes-agent/hermes_cli/profile_cmd.py` exposes native `profile show <name>` with an error for a missing profile. `profiles.py::get_profile_dir` maps `default` to the root, not `profiles/default`. Native profile selection must remain native: do not import private Python internals or duplicate home-resolution rules.
- Installed `plugins_cmd_listing.py` supports `plugins list --plain --no-bundled`: rows have status first and exact name last; pin annotations can contain spaces. Parse those fields, not display width, descriptions or substrings. The native update implementation explicitly refuses a pinned checkout. These local source observations determine the integration contract; do not modify Hermes upstream.
- [Previous task](../tasks/items/V0-011-e3.md), its [verified receipt](../tasks/evidence/V0-011-e3.toml), and [install documentation](install.md#pinned-installation-in-every-existing-profile) already record a successful all-profile installation at `c98d77ca34554efa2848767e707e3b6a631a7a5c`. Do not reopen that completed task or pretend this is the first install.
- Current local observations: `hermes` resolves through `~/.local/bin/hermes` to the existing Hermes environment; `HERMES_HOME` and `HERMES_PROFILE` are unset and no sticky-active marker exists, so the selected profile is `default`. Its research-forge metadata is pinned at that full revision. Installed plugin.yaml, __init__.py and methodology SKILL.md match this checkout byte-for-byte. The same pin is recorded for the named research-forge profile, but this request does not select all named profiles.
- `rforge version` reports the existing dev build at `a743265`; the read-only capability check for search/paper/brief/compare/provenance at the documented minimum succeeds. This supports preserving the binary, not an assertion that the entire current source tree is installed.
- The documented native `hermes skills list` enumerates filesystem skills and is not proof of plugin skill discovery. Runtime `skills_list` / `skill_view(name="research-forge:research-forge")` or the supported local registration API is the appropriate later qualification.
- Autowork capabilities report tasks prepare API 4. Store preparation reports READY, with pre-existing unrelated legacy-consumer/documentation and goal-without-work diagnostics. This slice neither migrates the queue nor repairs unrelated benchmark/goal code.

## Approach and planner's calls

1. **Chosen: native CLI lifecycle, no Hermes filesystem migration.** Keep Bash 3.2 compatibility and current installer flags. Let Hermes resolve current/explicit profiles. Remove Hermes's automatic standalone-skill quarantine; leave every existing standalone skill untouched and explain how the owner can inspect shadowing manually. Other hosts' quarantine behavior stays unchanged. This removes the unsafe path assumption instead of building another profile resolver.
2. **State verification:** use successful native `plugins list --plain --no-bundled` output and exact last-field equality to research-forge. Distinguish absent, installed-disabled, installed-enabled and native command failure. Do not treat a similarly named plugin, a help line or a failed probe as an installation. Before mutations, validate every explicitly requested profile through native `profile show NAME`; use `profile list` only for human-readable error choices, not as the validator. Current-profile selection keeps unqualified argv and environment intact.
3. **Idempotence/pins:** enabled existing research-forge is a reported no-op, preserving its source/revision; disabled existing research-forge may be enabled, never implicitly reinstalled. A missing plugin uses the existing isolated subdirectory URL and --no-deps, followed by explicit enable. Verify enabled exact identity afterward. Native install/enable/list failures remain nonzero with an actionable component error. An explicit update of a pinned install must fail clearly with native repin guidance and leave the pin unchanged; no --force, scanner override or automatic unpin.
4. **Local target:** explicitly qualify only default on this PC. Do not reuse the historical all-profile authorization as a new rollout. Because the plugin is already installed at the checkout's plugin revision, a correctly verified no-op is success; do not remove/reinstall just to demonstrate activity. The binary's current capability pass means no binary update is needed. If execution-time preflight differs, report it rather than silently broadening installation.
5. **No hidden MCP scope:** this remains a methodology-skill plugin installation. Hermes MCP registration is a separate manual workflow and is not added by this request.
6. **Authority:** the offline implementation task can be a complete Now record. Local installation/verification is a complete owner-owned live record in Needs decision, with live denied during standalone planning. A later explicit execution instruction is the transition to applying it on this PC; scan warnings still require the owner's decision for that exact source revision. No interview is needed to save either outcome.

Rejected alternatives: reconstruct Hermes home paths/import its private Python internals (duplicates host policy and breaks custom deployments); build new pin/source flags and an all-profile rollout (not needed for these observed defects and risks replacing the qualified pin); install immediately with the old script (violates this planning invocation and exercises the unsafe lifecycle).

## Offline implementation handoff

Expected write set: `install.sh`, `internal/installer/install_test.go` (Hermes mock fidelity only), new `internal/installer/hermes_safety_test.go`, `docs/install.md`.

Use the existing sandbox, injected HOME/PATH and fake native CLIs. No real Hermes imports, network, scanner bypass or machine profile access in tests. Begin with a failing `TestHermesPreservesStandaloneSkills` showing that the current implementation moves a legacy skill. Make the minimum production change, then test current/env-selected, sticky, explicit default/named and custom-home cases through native argv and preservation assertions. Do not hardcode new path resolution merely to satisfy a fixture.

`TestHermesIdempotentPinnedLifecycle` covers an enabled pinned exact plugin (no install/update/enable), a disabled existing plugin (enable only), a missing plugin (isolated subdir install then enable), and similarly named/disabled-after-enable output (not success). Native update refusal of a pin remains visible. Verify selected-profile state and unrelated profile/standalone bytes; neither a matching name nor return code from enable alone establishes success.

`TestHermesPreflightAndDryRun` covers failed native list/profile probes, a bad name in a multi-profile selection before any mutation, and dry-run preserving plugin/profile/skill state while showing the actual bounded native argv. Temporary installer scratch is not user/profile state. Existing `TestHermesDefaultProfileNeedsNoFlag`, explicit/unknown profile tests and non-Hermes installer tests must remain valid. Preserve existing command flags and do not introduce a dependency just for parsing plain rows.

Checks proposed, not run here: `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/installer -run TestHermes -count=1`; `bash -n install.sh`. The merged-tree full guard remains `make check`. This plan does not create the named test file or fix implementation.

## Local install/verification handoff

After the offline improvement is verified and later execution is explicitly authorized, use the landed checkout's Hermes-only installer path with explicit default selection. First retain a private, redacted before-state inventory: native exact plugin status/source pin and nonsecret file hashes for settings/standalone skills. Do not copy config contents, .env, credentials or personal runtime logs into the repository. Do not change other profiles or other host integrations.

A passing qualification records: the exact tested installer/source revision; preflight engine capability result; actual install/no-op/enable result; exact research-forge enabled status; unchanged full source pin and installed plugin file hashes; unchanged unrelated settings/skills; and actual runtime registration/loading of research-forge:research-forge. Use the supported local tools/registration path without a paid model research call. Do not launch/restart a gateway or bot. If runtime discovery cannot be checked safely without restarting an active user session, record that specific check as pending rather than claim a pass; safe plugin-state verification may still finish.

The qualification receipt belongs at `docs/qualification/hermes-local-installer.md`; it is a scoped local receipt, not four-host release qualification or a goal-met claim. The owner task's only repository write is this receipt. Real host plugin state is an explicit external effect of later execution, not something a worker can reproduce by changing a test fixture. No new standalone copy is created as a workaround for missing runtime discovery.

## Acceptance and pre-mortem

- Offline safeguards are demonstrated by the named deterministic sandbox tests, not a new scalar metric or a fabricated baseline. Local live observations must have actual command/result evidence; simulated scanner/plugin output is not machine qualification.
- Wrong profile/legacy skill mutation: preservation/argv tests guard current, explicit default and named/custom cases; Hermes-specific automatic quarantine is removed.
- False installed/enabled claim or silent pin movement: exact-row and pinned lifecycle tests guard negative names, disabled state, native failures and repeat install/update behavior.
- Unplanned rollout or security change: the local task is owner/live-gated, names one target, prohibits --force and policy/config/MCP changes, and records any remaining runtime check honestly.

No production, fixture, executable skill, service or user-configuration change is made by planning. Save this plan and the two task records, run the builder check and tasks validation, commit the planning batch once, then stop. Existing review/qualification records are retained; no task is closed or new verified evidence fabricated here.

## Saved handoff

- Installer improvement: `V0-013-2a4d9252`, Now, worker-owned, offline; next action is the failing standalone-skill preservation test.
- Installation on this PC: `V0-014-6706388b`, Needs decision, owner-owned, live denied, depends on the installer fix and later execution authority. Default-profile verification is the chosen scope, not another all-profile rollout.
- Fresh independent builder review: **nothing important left** beyond the explicit later local execution authorization. The real dependency is necessary; no accidental blocker, missing requested outcome or source/test ownership conflict was identified.
- Actual planning checks: existing installer shell syntax passes; the installed rforge capability probe passes; native source/metadata and installed file equality were inspected without invoking an install. `autowork tasks validate --worktree .` reports no drafts and store preparation reports READY. Unrelated legacy-consumer/documentation diagnostics remain as found; no scope expansion to repair them.
- New regression tests and real local installation/runtime qualification are proposed, not executed. A saved plan and a historical install receipt are not proof that the improved installer is implemented or newly qualified.
