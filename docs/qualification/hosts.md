# Per-host lifecycle qualification

Start here for **V-E5-run**, the opt-in release qualification of Claude Code,
Codex, Pi and Hermes. This is a procedure, not a record of completed live runs.
Copy the [receipt template](host-receipt-template.md) once per host and release.
Offline conformance does not qualify an installed host session.

The [release acceptance requirements](../verifiable-engine-plan.md#workstream-e--cross-host-packaging-added-2026-10-19)
require passive plugins, unchanged approval/permission/schedule settings and one
engine. Follow [agent installation](../agent-install.md) for ordinary use.

## 1. Prepare an isolated qualification account

Use a disposable OS account/VM, an empty working directory outside this repository,
and an authorized host account/model. Never run fault injection in a daily-use
profile or gateway. Do not weaken permissions, disable approval prompts, enable
schedules or accept an automatic binary/dependency installation to make a case pass.
Shell examples use Bash on Linux/macOS (or WSL); record the actual OS and shell.

Checklist, repeated for **each** host:

- [ ] Local install at OLD, upgrade to NEW (record refresh/replacement mode), then removal.
- [ ] Fresh published Git-source install at NEW, then removal.
- [ ] Missing engine, incompatible engine, MCP crash, interrupted-job resume.
- [ ] Before/after policy and schedule comparison at every stage.
- [ ] Leftover standalone skill detected and explicitly reported after removal.

Choose two real refs with different manifest versions and the native packaging
being qualified. Do not invent a release tag or count an update that stays on the
same version as an upgrade. If only one qualifying version exists, record the
transition as `pending_live`, not pass. Keep the engine version fixed during the
plugin upgrade; engine upgrades are separate, operator-owned actions.

```bash
# Set these to the selected release refs before continuing.
export OLD_REF='<older-tag-or-commit>' NEW_REF='<candidate-tag-or-commit>'
export REPO_URL=https://github.com/TrebuchetDynamics/research-forge.git
: "${TMPDIR:?set a disk-backed scratch directory}"
export RUN="$(mktemp -d "$TMPDIR/rforge-host-qualification.XXXXXX")"
export CHECKOUT="$RUN/checkout"
mkdir -p "$RUN/private" "$RUN/work"
chmod 700 "$RUN" "$RUN/private"
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_COMMON_DIR
unset GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES GIT_PREFIX
git clone "$REPO_URL" "$CHECKOUT"
git -C "$CHECKOUT" rev-parse "$OLD_REF^{commit}" "$NEW_REF^{commit}"
git -C "$CHECKOUT" show "$OLD_REF:package.json"
git -C "$CHECKOUT" show "$NEW_REF:package.json"
export OLD_SHA="$(git -C "$CHECKOUT" rev-parse "$OLD_REF^{commit}")"
export NEW_SHA="$(git -C "$CHECKOUT" rev-parse "$NEW_REF^{commit}")"
git -C "$CHECKOUT" checkout --detach "$OLD_SHA"
cd "$RUN/work"
node --version
rforge version
rforge capabilities --json --min-rforge-version 0.1.22 \
  --require search=1 --require paper=1 --require brief=1 \
  --require compare=1 --require provenance=1 --require mcp=1
export GOOD_PATH="$PATH" GOOD_ENGINE="$(command -v rforge)"
```

Save stdout, stderr, exit code and UTC time for every lifecycle command, plus
host `--version`, `plugin[s] --help` (Pi: `install --help`) and resolved install
path. If a host build lacks the documented native API, record its version and
mark that host case pending; a standalone skill fallback is **not** qualification.
Published-source installation must use the Git URL/shorthand below, not a local
clone or an npm archive. Record the actual fetched commit, not just `main`.

### Snapshot policy and detect shadow skills

Before installation, after installation, after upgrade, after each recovery and
after removal, capture the host's **effective** approval, permission/sandbox,
execution and scheduling settings in its settings UI or configuration output.
Include user, project and managed policy and environment overrides. Record the
schedule inventory even if empty; use `not supported` only with host-version
supporting evidence. Do not infer policy invariance merely from successful calls.

Use this helper to retain private file/directory snapshots, including absence.
Set `CONFIG_PATHS` using the host's list below and append any custom policy,
scheduling-extension or managed-policy paths used by the qualification account.
Run `snapshot before`, `snapshot installed`, `snapshot upgraded`,
`snapshot recovered` and `snapshot removed` at the corresponding stages (use a
unique suffix for each recovery and source mode).

```bash
snapshot() {
  local dest="$RUN/private/$1" i=0 p
  mkdir -p "$dest"
  for p in "${CONFIG_PATHS[@]}"; do
    printf '%s\n' "$p" > "$dest/$i.path"
    if [ -e "$p" ] || [ -L "$p" ]; then
      cp -pR "$p" "$dest/$i.value"
    else
      printf 'absent\n' > "$dest/$i.absent"
    fi
    i=$((i + 1))
  done
}
# Repeat with installed/upgraded/recovered snapshots, not only removed.
git diff --no-index "$RUN/private/before" "$RUN/private/removed"
```

`git diff --no-index` exits 1 when files differ: inspect the entire diff. The
only allowed semantic changes are ResearchForge package/marketplace bookkeeping,
explicitly operator-added/removed Hermes MCP wiring and the deliberate leftover
skill fixture. Every other config or schedule change fails the case. Record key
names and redacted before/after values, file hashes and absence markers. Do not
publish private snapshots, credentials, auth files or raw host logs. Even if a
host leaves an inert cache, it must have no active plugin/MCP/skill reference;
list retained cache paths and prove a new session does not load them. Restore
only ResearchForge-specific bookkeeping if the host leaves an inert disabled
entry, and record the manual cleanup; never overwrite an entire user config.

Run this inventory before install and after removal, from the empty work directory:

```bash
for root in "$HOME/.agents/skills" "$HOME/.claude/skills" \
  "$HOME/.codex/skills" "$HOME/.pi/agent/skills" "$HOME/.hermes/skills" \
  "$PWD/.agents/skills" "$PWD/.claude/skills" "$PWD/.codex/skills" \
  "$PWD/.pi/skills" "$PWD/.hermes/skills"; do
  if [ -e "$root/research-forge/SKILL.md" ]; then
    printf 'LEFTOVER standalone skill: %s\n' "$root/research-forge/SKILL.md"
  fi
done
```

Also inspect configured custom skill roots, symlinks and the selected Hermes
profile's `skills/`. Quarantine pre-existing standalone copies with operator
consent before installation; never remove parent skill directories or research.
To prove detection and preservation, **immediately before the final published-source
plugin removal** seed the former curl-install location in this disposable account:

```bash
export LEGACY="$HOME/.agents/skills/research-forge"
test ! -e "$LEGACY" && test ! -L "$LEGACY" || exit 1
mkdir -p "$LEGACY"
cp "$CHECKOUT/skills/research-forge/SKILL.md" "$LEGACY/SKILL.md"
# Record its hash, then run the host's final plugin removal commands.
# Repeat the inventory above AFTER removal: it must report this surviving path.
# Capture host skill discovery too: a remaining standalone skill is not a plugin.
# Only after recording survival/detection, clean up the independent fixture:
rm "$LEGACY/SKILL.md"
rmdir "$LEGACY"
```

The copy models the curl-installed file without downloading changing content.
Record its hash, detection output, possible shadow/duplicate discovery and cleanup;
repeat the inventory to prove absence. Plugin uninstall must not silently erase
an independently installed skill. Report a real leftover, never mask it as a
successful complete uninstall.

### Shared session probe (all hosts, both versions/source modes)

Create a tiny **synthetic**, agent-reported fixture; it is not engine-verified
research. Preserve it and the installed engine through removal.

```bash
export FIXTURE="$RUN/fixture"
mkdir -p "$FIXTURE"
printf '%s\n' '{"queries":["lifecycle fixture"],"sources":["fixture"],"results":1,"deduped":1,"failures":0}' > "$FIXTURE/manifest.json"
printf '%s\n' '{"title":"Synthetic lifecycle record","year":2026,"source":"fixture"}' > "$FIXTURE/results-deduped.jsonl"
rforge --json brief "$FIXTURE" > "$RUN/cli-brief.json"
```

Start the host from `$RUN/work`. Before any research request, inspect its plugin,
skill and MCP lists: exactly one ResearchForge integration; only methodology and
five explicit workflows (Hermes: methodology only); no `*-tdd` skills; no research
job, search request, hook or schedule from startup. MCP discovery/capability
probing at startup is allowed. Invoke the methodology without researching:
“Load ResearchForge methodology only; do not search, install anything or run tools.”
Then explicitly request the workflow probe named in each host section. Save the
loaded skill path and transcript; workflow loading must not itself execute a CLI
command. The agent may act only on the explicit request and normal host approval.

Through the host's native MCP tool interface, call `research_status` with
`{"dir":"<absolute FIXTURE path>"}` (replace the placeholder, do not pass a
literal `$FIXTURE`). Save tool arguments and raw structured result. Compare the
`{"ok":true,"data":...}` payload to `$RUN/cli-brief.json`, not the agent's prose.
Record discovered tool names: `search`, `job_status`, `source_get`,
`research_status`, `compare`, `evidence_retrieve`, `claim_check`. Approval,
claim acceptance and package export must not appear as tools. Other installed
servers' tools are outside this allowlist. Repeat this probe after upgrade and
recovery; artifact paths/status/evidence must not be rewritten by the adapter.

## 2. Claude Code

```bash
claude --version
claude plugin --help
CONFIG_PATHS=("$HOME/.claude/settings.json" "$HOME/.claude.json" \
  "$PWD/.claude/settings.json" "$PWD/.claude/settings.local.json" \
  "$PWD/.mcp.json")
snapshot before
# Local OLD install; marketplace selects adapters/, never --plugin-dir at root.
claude plugin marketplace add "$CHECKOUT"
claude plugin install research-forge@research-forge --scope user
claude plugin list
claude
```

In-session: `/reload-plugins`, `/mcp`,
`/research-forge:research-forge Load methodology only; do not research.` and
`/research-forge:brief Describe the workflow only; do not execute it.`
Run the shared probe; snapshot installed. Exit before updating the disposable clone:

```bash
git -C "$CHECKOUT" checkout --detach "$NEW_SHA"
claude plugin marketplace update research-forge
claude plugin update research-forge@research-forge --scope user
claude plugin list
claude
```

Reload/start a fresh session, prove OLD → NEW using installed manifest versions
and resolved source SHAs, rerun the probe and snapshot upgraded. Then remove:

```bash
claude plugin uninstall research-forge@research-forge --scope user
claude plugin marketplace remove research-forge
claude plugin list
```

Snapshot removed and start a new session: no namespaced skill/workflow or rforge
MCP server remains. In a fresh baseline, test the published source at NEW:

```bash
claude plugin marketplace add "TrebuchetDynamics/research-forge#$NEW_REF"
claude plugin install research-forge@research-forge --scope user
claude plugin list
claude
# After the shared probe and recovery cases, exit and repeat removal above.
```

**Receipt observations:** command results, user scope, marketplace path and
`adapters/.claude-plugin/plugin.json` version, OLD/NEW SHAs, `/mcp` registration
and tool payload, loaded skill names, startup inactivity, policy/schedule diffs,
new-session absence after removal and standalone-skill detection. Use matching
scope for every operation if separately qualifying project/local installation.

## 3. Codex

```bash
codex --version
codex plugin --help
CONFIG_PATHS=("$HOME/.codex/config.toml" "$HOME/.agents/plugins/marketplace.json" \
  "$PWD/.codex/config.toml" "$PWD/.agents/plugins/marketplace.json")
snapshot before
codex plugin marketplace add "$CHECKOUT"
codex plugin add research-forge@research-forge
codex plugin marketplace list
codex
```

In-session, invoke `$research-forge:research-forge Load methodology only; do not research.`
and `$research-forge:brief Describe the workflow only; do not execute it.` These
are Codex skill invocations, not shell variables or Hermes slash commands.
Inspect MCP tools in the session and run the shared probe; snapshot installed.
Exit, update the disposable local checkout and replace the installed cache through
the supported remove/add commands. Record this explicitly as a replacement upgrade,
not an atomic in-place update; preserve research and all non-plugin config:

```bash
git -C "$CHECKOUT" checkout --detach "$NEW_SHA"
codex plugin remove research-forge@research-forge
codex plugin add research-forge@research-forge
codex plugin marketplace list
codex
```

`codex plugin marketplace upgrade research-forge` refreshes **Git** snapshots,
not local-path marketplaces; do not use it for this local upgrade. A restart alone
is not proof that an installed cache picked up the new checkout.

Verify the **loaded** `adapters/.codex-plugin/plugin.json` changed from OLD to NEW,
not just the catalog. Rerun the probe and snapshot upgraded. Remove and check a
new session, then take a fresh baseline and install the published source:

```bash
codex plugin remove research-forge@research-forge
codex plugin marketplace remove research-forge
codex plugin marketplace list
# Snapshot removed; check absence in a new session before proceeding.
codex plugin marketplace add TrebuchetDynamics/research-forge --ref "$NEW_REF"
codex plugin add research-forge@research-forge
codex plugin marketplace list
codex
# After the shared probe and recovery cases, exit and repeat removal above.
```

**Receipt observations:** native plugin support/version (fallback skills do not
count), marketplace snapshot and loaded cache paths/SHAs/versions, one methodology
plus five explicit-only workflows, shared MCP results, no automatic work,
approval/sandbox/schedule before/after, no active plugin/config/MCP reference on
removal, retained cache disposition and leftover-skill report. Do not compensate
for managed-policy restrictions by changing approval policy.

## 4. Pi

Requires Pi 1.3.0 or a build with native MCP and `registerMcpServer()`; Node 22+.
Do not install `pi-mcp-adapter` or add an `mcp.json` entry for this package.

```bash
pi --version
pi install --help
CONFIG_PATHS=("$HOME/.pi/agent/settings.json" "$HOME/.pi/agent/mcp.json" \
  "$PWD/.pi/settings.json" "$PWD/.pi/mcp.json")
snapshot before
pi install "$CHECKOUT"
pi list
pi
```

In-session: `/reload`, `/mcp`,
`/skill:research-forge Load methodology only; do not research.` and
`/rforge_brief Describe the workflow only; do not execute it.` Verify all five
`/rforge_` commands and workflow skills are discoverable. Run the shared probe
using `mcp__rforge__research_status`; snapshot installed. Exit, then:

```bash
git -C "$CHECKOUT" checkout --detach "$NEW_SHA"
pi update --extensions
pi list
pi
```

Reload/start fresh, verify the loaded `package.json` OLD → NEW version, same local
source path and only one MCP registration; repeat the probe and snapshot upgraded.
`pi mcp list` is **not** the session probe: shell MCP commands do not load extensions.
A user/project `mcp.json` entry named `rforge` overrides this extension; report it
and use a clean disposable profile, not a second server or modified permissions.

```bash
# Use the exact local source printed by pi list (normally CHECKOUT).
pi remove "$CHECKOUT"
pi list
# Snapshot removed; check absence in a new session, then a fresh baseline:
pi install "git:github.com/TrebuchetDynamics/research-forge@$NEW_REF"
pi list
pi
# After the shared probe and recovery cases, exit; use exact source from pi list:
pi remove "git:github.com/TrebuchetDynamics/research-forge@$NEW_REF"
```

**Receipt observations:** Pi/Node versions, package source/scope and installed
commit/version, native `/mcp` source and tool results, commands loading instructions
without shell interpolation, queued follow-up behavior if invoked while busy,
no startup research, unchanged policy/schedules, no persisted MCP file from the
extension, no active tools/skills after reload/removal and leftover-skill report.
Personal scope is used here; match `--local` on install/remove for a project case.
Pinned Git refs do not advance with `pi update --extensions`; to move a pin use
`pi remove <exact-old-source>` then `pi install <new-source>`, recording this as
pin replacement. The local OLD → NEW case above supplies the in-place upgrade.

## 5. Hermes

Use a dedicated profile for **every** command. Set `PROFILE` and the absolute
`HERMES_PROFILE_HOME` to the existing disposable profile and verify them in the
host before taking snapshots; do not assume the global default profile.

```bash
export PROFILE='<qualification-profile>'
export HERMES_PROFILE_HOME='<absolute-selected-profile-directory>'
hermes --version
hermes --profile "$PROFILE" plugins --help
CONFIG_PATHS=("$HERMES_PROFILE_HOME/config.yaml" "$HERMES_PROFILE_HOME/cron")
snapshot before
# file:// clones a local Git checkout; a bare filesystem path is not a Git URL.
hermes --profile "$PROFILE" plugins install "file://$CHECKOUT" --ref "$OLD_SHA"
hermes --profile "$PROFILE" plugins enable research-forge
hermes --profile "$PROFILE" plugins list
```

Decline the Node dependency prompt caused by Pi's `package.json`; no npm packages
are needed. Record the installed plugin path from `plugins list` as `PLUGIN_DIR`.
**Manually** merge only this server into the selected profile's `config.yaml`,
substituting the absolute installed path (not the checkout path):

```yaml
mcp_servers:
  rforge:
    command: node
    args: ["<absolute-PLUGIN_DIR>/adapters/mcp.mjs"]
```

Preserve other servers and policy keys. Save the exact redacted diff as an
operator action, separate from plugin-caused changes. The plugin only registers
a read-only skill; enabling it must not add this server or edit runtime defaults.

```bash
hermes --profile "$PROFILE" mcp test rforge
hermes --profile "$PROFILE"
```

Restart the dedicated profile's gateway/session as required by that build; do
not restart a shared production gateway. Ask for
`skill_view(name="research-forge:research-forge")` only, without research. Hermes
has no ResearchForge workflow slash commands. Run the shared MCP probe; snapshot
installed. Exit, then explicitly advance the pinned local-source install:

```bash
git -C "$CHECKOUT" checkout --detach "$NEW_SHA"
hermes --profile "$PROFILE" plugins install "file://$CHECKOUT" --force --ref "$NEW_SHA"
hermes --profile "$PROFILE" plugins list
hermes --profile "$PROFILE" mcp test rforge
hermes --profile "$PROFILE"
```

Verify OLD → NEW `plugin.yaml` version and full installed commit, enabled state,
manual MCP path validity, preserved operator config, skill and tool probe after
restart; snapshot upgraded. A pinned install deliberately will not advance with
`plugins update`; forced same-source re-pin is the explicit upgrade operation.

```bash
hermes --profile "$PROFILE" plugins disable research-forge
hermes --profile "$PROFILE" plugins remove research-forge
# Manually remove only the mcp_servers.rforge entry added above.
hermes --profile "$PROFILE" plugins list
hermes --profile "$PROFILE"
# Check a new session, snapshot removed, then take a fresh baseline:
hermes --profile "$PROFILE" plugins install "$REPO_URL" --ref "$NEW_SHA"
hermes --profile "$PROFILE" plugins enable research-forge
hermes --profile "$PROFILE" plugins list
# Re-add the manual MCP entry using the new installed path, test and restart.
# After the probe/recovery cases, repeat disable/remove and manual MCP cleanup.
```

The published Git pin uses a **full commit SHA**, not a tag. For an unpinned
published installation, the native refresh command is
`hermes --profile "$PROFILE" plugins update research-forge`; do not report a no-op
as a version transition. Check the launcher path after any update.

**Receipt observations:** profile/home, host version, install/enable consent,
manifest version/source SHA, one namespaced methodology skill (no copied profile
skill or workflows), no `defaults.json`/hooks/schedules/dependency installation,
manual MCP add/remove diffs, `mcp test` and session results, unchanged approvals,
permissions, schedule inventory and runtime defaults, enabled/disabled bookkeeping
cleanup, no residual MCP reference and standalone skills in the selected profile
as well as common roots.

## 6. Recovery matrix — execute for each installed host

Use the same session/probe/receipt rules for **every** host; run on the NEW
published-source install before final removal. Never inject faults into the
engine used by another host/session. Set `HOST_BIN` to the absolute host executable
(`command -v claude`, `codex`, `pi` or `hermes`), and use the host-specific launch
row below. Restart a dedicated Hermes gateway with the same fault environment if
it owns MCP; changing only a connecting client's PATH will not test its engine.

| Host | Launch under fault environment | Inspect / reconnect after repair |
| --- | --- | --- |
| Claude Code | `env PATH="$FAULT_PATH" "$HOST_BIN"` | `/mcp`, reconnect rforge; `/reload-plugins` or new session |
| Codex | `env PATH="$FAULT_PATH" "$HOST_BIN"` | session MCP tools/errors; exit and start a new session |
| Pi | `env PATH="$FAULT_PATH" "$HOST_BIN"` | `/mcp`, `/mcp reconnect rforge` or `/reload` |
| Hermes | `env PATH="$FAULT_PATH" "$HOST_BIN" --profile "$PROFILE"` | `hermes --profile "$PROFILE" mcp test rforge`, restart dedicated gateway/session or `/reload-mcp` |

### R1 — rforge missing

Construct `NO_ENGINE_PATH`, a path containing Node, the host and its runtime
helpers but **no** rforge (including symlinks). Do not rename/delete the real
binary. Verify the actual spawn environment before setting `FAULT_PATH`:

```bash
export NO_ENGINE_PATH="$RUN/no-engine"
python3 - <<'PY'
import os
from pathlib import Path
out = Path(os.environ['NO_ENGINE_PATH'])
out.mkdir()
for directory in os.get_exec_path():
    root = Path(directory or '.').absolute()
    if not root.is_dir():
        continue
    for executable in root.iterdir():
        target = out / executable.name
        if (executable.name != 'rforge' and not target.exists()
                and executable.is_file() and os.access(executable, os.X_OK)):
            target.symlink_to(executable)
PY
env PATH="$NO_ENGINE_PATH" node -e 'const {spawnSync}=require("node:child_process"); const r=spawnSync("rforge",["version"]); if(r.error?.code!=="ENOENT") process.exit(1)'
export FAULT_PATH="$NO_ENGINE_PATH"
```

Launch with the host row, inspect the failed MCP connection and attempt the shared
probe. **Record:** JSON-RPC `error.data.ok: false`, code `rforge_missing`,
`error.install: "install.sh"`, actionable install/upgrade/reconnect message,
no research artifacts and no automatic download. If the UI hides structured data,
attach the redacted MCP diagnostic log too. Host startup failing for a missing
runtime helper does not satisfy this case. Exit; relaunch with `GOOD_PATH`,
reconnect and record the successful identical fixture result and policy snapshot.

### R2 — wrong version / incompatible engine

Build a real engine with a deliberately too-old release label in scratch (Go
1.26+). This tests version negotiation, not behavior of a historical binary;
record that distinction. Capability gaps/old engines without `capabilities` may
additionally produce `rforge_incompatible` rather than `version_too_old`.

```bash
mkdir -p "$RUN/old-engine"
(cd "$CHECKOUT" && go build -ldflags '-X main.version=0.1.0' \
  -o "$RUN/old-engine/rforge" ./cmd/rforge)
"$RUN/old-engine/rforge" version
export FAULT_PATH="$RUN/old-engine:$GOOD_PATH"
```

Launch with the host row and request the probe. **Record:** rejected version,
minimum/capability diagnostics, `version_too_old` for this build, structured false
result and `install.sh` recovery hint; no search or auto-upgrade. Exit, restore
`GOOD_PATH`, reconnect, repeat the probe and snapshot policy. Reinstalling the
plugin is not a repair for a missing/incompatible engine.

### R3 — MCP server crash

Under `GOOD_PATH`, pass the shared probe. Identify the **specific session's**
`rforge mcp serve` child with `ps -eo pid,ppid,args`; save its PID, parent chain and
command, and confirm it belongs to this disposable host before proceeding:

```bash
export MCP_PID='<verified-session-rforge-mcp-serve-pid>'
ps -p "$MCP_PID" -o pid,ppid,args
kill -KILL "$MCP_PID"
```

Do not use `pkill`, `killall` or kill the host/other sessions. Request the fixture
probe again; record the visible disconnect/tool error or host-owned reconnect,
not a fabricated success or a hung call. Use the host row's reconnect/restart;
record new server PID, same fixture result, no duplicate tool registration,
unchanged policy and no orphaned old server. Do not require a particular host's
automatic reconnect policy.

### R4 — interrupted job resumes from persisted state

This case intentionally uses live OpenAlex metadata (no paid source, PDFs or
exports); record endpoint, time and request limits. Start the dedicated host/MCP
owner with the following environment, then use its recovery-matrix launch row,
so there is time to interrupt between pairs. This is temporary process environment,
not a saved host scheduling setting:

```bash
export RFORGE_SOURCE_REQUEST_DELAY=5s FAULT_PATH="$GOOD_PATH"
```

Call native MCP `search` exactly once with:

```json
{"dir":"<RUN>/interrupted","queries":["artificial photosynthesis","photoelectrochemical water splitting","solar hydrogen catalysts"],"sources":["openalex"],"limit":1,"depth":"standard","max_requests":6,"continue_on_error":true}
```

Replace `<RUN>` with the actual absolute path. Record `data.job_id` as `JOB_ID`.
Poll native `job_status` with `{"dir":"<RUN>/interrupted","id":"<JOB_ID>"}`.
Wait for `0 < progress.completed < progress.total`, save the committed result
prefix and checkpoint, then use R3's verified PID command to crash this server.
If it already completed, use a new directory and repeat; a completed job is not
an interruption. Do not create another search to simulate resumption.

```bash
export JOB_DIR="$RUN/interrupted" JOB_ID='<returned-job-id>'
# Capture these BEFORE the crash once at least one pair has committed.
cp "$JOB_DIR/batch-state.json" "$RUN/private/checkpoint-before.json"
cp "$JOB_DIR/results.jsonl" "$RUN/private/results-before.jsonl"
# After killing the verified server:
rforge --json jobs status "$JOB_ID" --dir "$JOB_DIR"
rforge --json jobs resume "$JOB_ID" --dir "$JOB_DIR"
rforge --json jobs status "$JOB_ID" --dir "$JOB_DIR"
rforge --json brief "$JOB_DIR"
```

First status must persist `interrupted` after the dead writer's lease is released.
Resume is deliberately CLI/human-gated: there is **no MCP resume tool**. Record
same job ID, saved queries/options, before/after completed pairs, cumulative
requests used, stop reason, results/manifest/provenance paths and hashes. Completed
pairs must be skipped, not repeated or silently lost; compare the saved JSONL
prefix and pair checkpoint with the final artifacts. A partial budget/error stop
is not proof of a completed resume; report remaining work and failures explicitly.
Do not reset the budget/checkpoint or delete a lock to force success.

Reconnect/restart the host, call `job_status` for the **same ID/directory**, then
`research_status` for that directory. Compare raw payloads to the final CLI status
and brief, preserving evidence identities, provenance and failure details; normalize
only timestamps and nondeterministic IDs for separate executions. Capture the
checkpoint/result diff and confirm no duplicate records from replayed pairs.
A host/tool-call cancellation alone need not stop a job: work belongs to the MCP
server's lifetime. Run `unset RFORGE_SOURCE_REQUEST_DELAY` before the next launch,
snapshot policy, then seed the independent leftover skill, perform the host's final uninstall,
and verify leftover survival/detection before its explicit cleanup.

## 7. Evidence and maintenance

Complete every receipt row as `pass`, `fail` or `pending_live`; never promote an
offline conformance run or unexecuted command to a live pass. Link redacted evidence
and exact commands. Unexpected policy mutation, silent installs, stale loaded
versions, leaked developer skills or non-resumable jobs are failures. Host/model
access, unavailable release pairs and live-service outages remain explicit pending
live work. V-E5 writes this procedure; **V-E5-run** executes it per release.

Source anchors (reviewed against this checkout; update this page when they change):

- [Manifests and passive packaging gate](../../internal/docs/plugin_conformance_test.go)
  (`plugin_conformance_test.go:20`, `TestPassivePluginConformance`) verifies versions,
  curated skill roots, explicit-only workflows and prohibited runtime surfaces.
- [Shared preflight](../../adapters/mcp.mjs): `mcp.mjs:26` (`compatibility`),
  `mcp.mjs:52` (`serveFailure`) and signal handling;
  [Pi registration](../../adapters/pi.mjs): `pi.mjs:13` (`researchForge`).
- [Hermes registration](../../__init__.py): `__init__.py:6` (`register`);
  [manifest](../../plugin.yaml).
- [Engine MCP lifecycle](../../internal/mcp/server.go): `server.go:71` (`Serve`),
  `search`, `job_status`, `research_status`;
  [job CLI](../../internal/cli/jobs_command.go): `jobs_command.go:16` (`executeJobs`);
  [brief fixture format](../../internal/research/brief.go): `brief.go:51` (`Brief`).
- [Offline lifecycle evidence](../../internal/mcp/lifecycle_test.go):
  `lifecycle_test.go:24` (`TestSearchLifetimeBelongsToServer`);
  [launcher parity/recovery tests](../../adapters/test/engine.test.mjs).
- Host command references: [Claude lifecycle CLI](https://code.claude.com/docs/en/plugins/cli-reference),
  [Claude marketplace refs](https://code.claude.com/docs/en/plugins/host-marketplace),
  [Codex marketplace packaging](https://developers.openai.com/plugins/build/plugins),
  [Pi packages](https://pi.dev/docs/latest/packages),
  [Pi native MCP](https://pi.dev/docs/latest/mcp),
  [Hermes plugins and pins](https://hermes-agent.nousresearch.com/docs/user-guide/features/plugins),
  [Hermes Git/local URL parsing](https://github.com/NousResearch/hermes-agent/blob/main/hermes_cli/plugins_cmd.py).

Requalify on changes to host versions/APIs, manifests, launcher minimum or
capabilities, tool schemas/lifecycle, skill discovery, job persistence or any
installation command. Receipts must identify the source revision of this runbook.
