# Host lifecycle qualification receipt — TEMPLATE (not executed)

Copy once per host/release. Follow [the runbook](hosts.md). Replace placeholders;
retain explicit `pending_live` entries for anything unexecuted. Do not attach
credentials, private configuration snapshots or unredacted transcripts.

## Identity and reproduction

| Field | Recorded value |
| --- | --- |
| Host (Claude Code / Codex / Pi / Hermes) and exact version | |
| Operator/reviewer, UTC start/end, receipt ID | |
| OS/architecture/shell; disposable account/VM identifier | |
| Runbook source commit; candidate release | |
| Local checkout absolute path; clean-tree evidence | |
| OLD ref / full commit SHA / manifest version | |
| NEW ref / full commit SHA / manifest version (must differ from OLD) | |
| Published Git source string / resolved installed SHA | |
| Actual loaded install/cache path and manifest, before/after upgrade | |
| Install scope; project directory; profile/home (Hermes) | |
| Host executable path; Node version/path; Go version for R2 | |
| Engine path / version / capability JSON / binary SHA-256 | |
| Model/account authorization (no secrets); relevant managed policy | |
| Additional config/skill/schedule roots and environment overrides | |
| Redacted evidence directory or immutable artifact links | |

## Command log

Include exact expanded argv, working directory, source mode, UTC time, exit code,
stdout/stderr artifact and reload/restart action for every command. Interactive
commands/tool calls need a transcript reference and raw structured result.

| Stage / source mode | Exact command or MCP name + JSON arguments | UTC / exit | Redacted evidence |
| --- | --- | --- | --- |
| Preflight, host support, engine capabilities | | | |
| Local OLD install / enable / restart | | | |
| Local OLD → NEW update / restart | | | |
| Local removal / new-session inspection | | | |
| Fresh published Git NEW install / restart | | | |
| R1 missing / repair / reconnect | | | |
| R2 incompatible / repair / reconnect | | | |
| R3 crash / reconnect | | | |
| R4 search / interruption / CLI resume / host readback | | | |
| Published removal / manual cleanup / new-session inspection | | | |
| Leftover seed before removal / survival + detection after removal / cleanup / absence | | | |

## Case results

Use only `pass`, `fail`, `pending_live`. Each pass requires linked observations,
not merely a command returning zero. A same-version update is not an upgrade.

| Required case | Result (initially pending_live) | Observed value / evidence / failure or pending reason |
| --- | --- | --- |
| Local checkout OLD install; loaded SHA/version | pending_live | |
| Local NEW upgrade; refresh/replacement mode and different loaded SHA/version | pending_live | |
| Fresh published Git-source NEW install; loaded SHA/version | pending_live | |
| Only public skills/workflows; no developer TDD skill | pending_live | |
| Methodology loads, explicit workflow only loads instructions (Hermes: no workflows) | pending_live | |
| Startup does no research, downloads, hooks or schedules; preflight only | pending_live | |
| Native MCP registration and exact tool allowlist | pending_live | |
| Fixture raw MCP envelope equals CLI JSON, OLD and NEW | pending_live | |
| R1: genuine PATH ENOENT; rforge_missing and install.sh hint; repair succeeds | pending_live | |
| R2: deliberately old-labelled engine rejected; version_too_old; repair succeeds | pending_live | |
| R3: verified session server PID crashed; visible failure/reconnect, probe restored | pending_live | |
| R4: same durable job interrupted, resumed without repeating completed pairs | pending_live | |
| R4: final host job_status/research_status equals CLI status/brief | pending_live | |
| Approval/permission/execution settings unchanged at every stage | pending_live | |
| Schedule inventory/settings unchanged at every stage | pending_live | |
| Local remove: config clean, new session has no plugin/skills/tools | pending_live | |
| Published remove: config clean, new session has no plugin/skills/tools | pending_live | |
| Research artifacts and separately installed engine preserved | pending_live | |
| Standalone curl-style skill seeded before removal survives, is detected/reported, then explicitly cleaned | pending_live | |

### Host-specific evidence (complete the applicable row)

| Host | Required observations and evidence |
| --- | --- |
| Claude Code | Marketplace selects adapters/; user/specified scope consistent; namespaced methodology + five workflows; /mcp server source; installed manifest versions; no active enabledPlugins/marketplace/MCP reference after removal. |
| Codex | Native plugin API, not skill fallback; catalog AND loaded cache versions/SHAs; $research-forge: invocation; one methodology + five explicit workflows; no active plugin/marketplace/MCP config after removal. |
| Pi | Node 22+, native MCP API; exact pi list source/scope; five /rforge_ commands and six skills; queued busy invocation if tested; native /mcp registration source; no newly persisted mcp.json; removal uses matching source and --local scope where applicable. |
| Hermes | Selected profile/home; enable consent; one research-forge:research-forge skill, no copied profile skill/workflows/defaults; Node dependency prompt declined; operator-only MCP add/remove diff and installed absolute launcher path; mcp test; re-pin preserves state; no residual active enabled/disabled/MCP entry after documented cleanup. |

## Policy and schedule invariance

Retain private snapshots locally; publish only hashes, key paths and redacted
semantic diffs. Compare **effective** policy as well as files. Record absent files
as absent, not empty. Cover project/user/managed policies and custom extensions.
Allowed deltas are only ResearchForge install bookkeeping, explicitly logged
Hermes MCP wiring and the deliberate standalone-skill fixture. Do not use those
exceptions to hide approval, sandbox, execution, schedule or runtime-default edits.

| Stage (repeat for each source and recovery) | Config paths / before→after hashes or absence | Effective approval / permission / execution before→after | Schedule paths + inventory before→after | Allowed bookkeeping/operator diff; evidence |
| --- | --- | --- | --- | --- |
| Before local install → installed | | | | |
| Installed → upgraded | | | | |
| Upgraded → local removed | | | | |
| Before published install → installed | | | | |
| R1 fault → repaired | | | | |
| R2 fault → repaired | | | | |
| R3 crash → recovered | | | | |
| R4 interrupted → resumed | | | | |
| Published installed → removed | | | | |
| Leftover seeded → detected → cleaned | | | | |

Unexpected mutation: `<none, or exact key/path and case failure>`.
Retained inert cache/disabled bookkeeping, manual cleanup and new-session absence
proof: `<paths, reason, exact narrow cleanup and evidence>`.
Hermes manual MCP edits (if applicable): `<redacted added/removed entry diff>`.

## Recovery details

- **R1:** host/MCP-owner PATH (redacted as needed), Node/host available, rforge
  spawn error ENOENT, raw JSON-RPC code/data/install hint, no download/artifacts,
  GOOD_PATH restoration and successful fixture result: `<evidence>`.
- **R2:** build command, binary hash, rejected version/minimum/capabilities;
  explicitly distinguish a deliberately old-labelled current engine from a
  historical release binary; error payload, no auto-upgrade, successful repair:
  `<evidence>`.
- **R3:** verified server PID/parent command, kill timestamp/signal, failure or
  automatic reconnect observation, manual reconnect/restart if used, new PID,
  restored probe, no duplicates/orphaned server: `<evidence>`.
- **R4:** source endpoint/time/process delay, exact search arguments/limits,
  job ID/directory, saved job inputs, before/crash/final status, progress pairs,
  stop reason and cumulative requests, saved checkpoint/result prefix hashes and
  comparison, CLI resume argv/exit, output/manifest/provenance references/hashes,
  raw final CLI/MCP parity evidence, duplicates/loss/failures: `<evidence>`.
- No claim of engine-verified provenance for the synthetic fixture. Record actual
  provenance levels and evidence identities; do not normalize away differences
  except allowed timestamps/nondeterministic IDs for separate runs.

## Removal and standalone-skill audit

| Observation | Local removal | Published removal / final audit |
| --- | --- | --- |
| Plugin/package and marketplace list after removal | | |
| New session: no plugin workflow/skill/MCP registration | | |
| Config/reference absence; manual cleanup if needed | | |
| Engine binary unchanged / still usable | | |
| Fixture/research/job artifacts retained and hashes checked | | |
| Common, custom and profile-local standalone skill inventory | | |
| Leftover SKILL.md before/after uninstall hash, survival and detection output | | |
| Shadow/duplicate discovery or absence; not confused with plugin survival | | |
| Explicit leftover cleanup and repeated absence inventory | | |

## Verdict and follow-up

- Overall: `pending_live` (change only after reviewing every required row).
- Failed cases and exact observed deviations: `<none or list>`.
- Pending live cases, host/service/access requirement and next command: `<list>`.
- Unexecuted/unsupported surfaces and reason (never silently skip): `<list>`.
- Offline conformance commands/results, **not live qualification**: `<list>`.
- Reviewer, UTC review time, redacted evidence links and release disposition:
  `<not yet reviewed>`.
