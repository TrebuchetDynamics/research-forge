# Agent installation

ResearchForge provides passive plugins for Claude Code, Codex, Pi and Hermes over
the same standalone `rforge` engine. Install one integration per host. The
`research-forge` methodology skill remains model-invocable when research is relevant;
developer `*-tdd` skills are never exposed. Claude Code, Codex and Pi also expose
five explicit-only workflows: `lit`, `paper`, `brief`, `compare` and `replicate`.
Hermes currently exposes only the methodology skill.

Workflow skills have `disable-model-invocation: true` and Codex policies with
`allow_implicit_invocation: false`. They name the CLI call and defer methodology
to `research-forge`; they contain no research logic. Pi commands load these
instructions on invocation. Typed engine operations use the same stdio MCP server
on every host: `rforge mcp serve`, behind the compatibility-checking
[`adapters/mcp.mjs`](../adapters/mcp.mjs) launcher. Claude Code and Codex register
it in their plugin manifests; Pi uses its built-in MCP client. Hermes requires
one manual config entry below; the plugin never edits user configuration.

The host may connect and discover tools when a session starts. This performs a
read-only compatibility check, not research. Research runs only on tool invocation.
There are no hooks, schedules, approval/permission changes or runtime defaults.
Approval, claim acceptance and package export remain CLI/human-gated, not tools.
All MCP hosts require Node 22+ and `rforge` on the host process's `PATH`.

## Install the engine separately

Plugins do **not** download or install the binary. Install or upgrade it yourself:

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/research-forge/main/install.sh | bash
rforge version
rforge capabilities --json --min-rforge-version 0.1.22 \
  --require search=1 --require paper=1 --require brief=1 \
  --require compare=1 --require provenance=1 --require mcp=1
```

A released engine must meet the minimum version **and** required capabilities.
A `dev` engine is accepted only on verified capabilities, not its version label.
Every host launcher runs this check before starting its first MCP session/tool
call, with a 10-second hard timeout and no shell. Missing or incompatible engines
fail closed with a structured JSON-RPC error containing `error.data.ok: false`,
the engine error code (or `rforge_missing` / `rforge_incompatible`) and
`error.data.error.install: "install.sh"`. Nothing is downloaded automatically.
If the check fails (including an older engine without `capabilities`), upgrade
and reconnect the MCP server before researching. With Go 1.26+, source installation is also
available: `go install github.com/TrebuchetDynamics/research-forge/cmd/rforge@latest`.
See [engine installation](install.md) for platforms and installer options.

## Migrate a curl-installed skill first

Before installing a plugin, **delete the old standalone `research-forge` skill
copy** from the host's user and project skill directories. Common locations are
`~/.agents/skills/research-forge`, `~/.claude/skills/research-forge`,
`~/.codex/skills/research-forge`, `~/.pi/agent/skills/research-forge` and
`~/.hermes/skills/research-forge` (or the selected Hermes profile's `skills/`).
Check project-local `.agents/skills/`, `.claude/skills/` and `.pi/skills/` too.
Remove only the old ResearchForge skill, not the parent skills directory or any
research artifacts. Leaving it behind can shadow the plugin or duplicate discovery.

## Claude Code

Requires Claude Code with plugin marketplace support.

```sh
claude plugin marketplace add TrebuchetDynamics/research-forge
claude plugin install research-forge@research-forge
```

For a local checkout, replace the marketplace source with `.` from its root.
Start a new session (or `/reload-plugins`) and invoke
`/research-forge:research-forge <question>`, or ask for the research-forge skill.
For an explicit workflow, use `/research-forge:lit <request>` (or `paper`, `brief`,
`compare`, `replicate` in place of `lit`).

Upgrade and remove:

```sh
claude plugin marketplace update research-forge
claude plugin update research-forge@research-forge
# Removal:
claude plugin uninstall research-forge@research-forge
claude plugin marketplace remove research-forge
```

The default install scope is user. If you used `--scope project` to install, use
that same scope when uninstalling. Do not load the repository root with
`--plugin-dir`: the curated plugin root is `adapters/`.

## Codex

Requires a Codex CLI with `codex plugin` support (not just standalone skills).

```sh
codex plugin marketplace add TrebuchetDynamics/research-forge --ref main
codex plugin add research-forge@research-forge
```

For a local checkout use `codex plugin marketplace add .` instead. Start a new
session and invoke `$research-forge:research-forge <question>`, or ask Codex to
use the methodology skill. Invoke a workflow explicitly with
`$research-forge:lit <request>` (or `paper`, `brief`, `compare`, `replicate`).
There is no Hermes-style slash command.

Upgrade the Git marketplace snapshot, then start a new session:

```sh
codex plugin marketplace upgrade research-forge
# Removal:
codex plugin remove research-forge@research-forge
codex plugin marketplace remove research-forge
```

For local installs, update the checkout and restart the client. If your Codex
build does not support plugins, use the generic skill fallback below.

## Pi

Requires Pi package support, its built-in MCP client and `registerMcpServer()`
(verified with Pi 1.3.0), plus Node 22 or later. Older Pi builds without that API
must be upgraded; this package does not install a third-party MCP client.

```sh
pi install git:github.com/TrebuchetDynamics/research-forge
# Or from a local checkout (keep the checkout in place):
# pi install .
```

Use `/reload` or start a new session, then
`/skill:research-forge <question>`. Native commands load the corresponding
explicit-only workflow skill and pass your request unchanged to the host agent:

| Command | Engine call described by the skill |
| --- | --- |
| `/rforge_lit <request>` | `rforge lit --research-dir <dir> --out <dir>` |
| `/rforge_paper <request>` | `rforge paper <identifier-or-title>` |
| `/rforge_brief <request>` | `rforge brief <search-dir> --out <brief.md>` |
| `/rforge_compare <request>` | `rforge compare <dir-a> <dir-b> --out <comparison.md>` |
| `/rforge_replicate <request>` | `rforge replicate <doi-or-arxiv-id> --out <plan.md>` |

For example: `/rforge_brief summarize research/topic into research/brief.md`.
These commands do not execute the engine themselves or interpret your request
as shell arguments. While the agent is busy, the request queues as a follow-up.
Pi also supports explicit `/skill:lit`, `/skill:paper`, `/skill:brief`,
`/skill:compare` and `/skill:replicate` invocation. The methodology skill keeps
its portable `/skill:research-forge` name.

Separately, the extension registers the checked `rforge` server with Pi's native
MCP client, without writing `mcp.json`. Inspect it with `/mcp`. The engine owns
schemas and results; there is no duplicate CLI-tool implementation in Pi.
The MCP v1 read/run tools are `search`, `job_status`, `source_get`,
`research_status`, `compare`, `evidence_retrieve` and `claim_check`.
`claim_check` verifies stored evidence facets; it cannot approve or accept a claim.
Pi names these tools `mcp__rforge__<tool>` (for example,
`mcp__rforge__research_status`). A user-configured server named `rforge` overrides
the plugin registration: remove an obsolete override rather than installing a
second copy.

```sh
pi list
pi update --extensions
# Removal: use exactly the source shown by pi list.
pi remove git:github.com/TrebuchetDynamics/research-forge
```

Use the corresponding local path with `pi remove` for a local install. If you
installed with `--local`, use the same flag for removal. Pinned Git tags/commits
do not advance during updates; remove the old source and install the new ref to
upgrade a pin. Unpinned Git installs track the upstream branch.

## Hermes

Requires Hermes native plugins with `register_skill` support. Use the intended
profile for each command (for example, `hermes --profile <name> plugins ...`).

```sh
hermes plugins install https://github.com/TrebuchetDynamics/research-forge
hermes plugins enable research-forge
```

Restart the Hermes session or gateway. Ask Hermes to load
`skill_view(name="research-forge:research-forge")` and research your question.
The entry point registers only this read-only skill: it does not copy skills
into the profile, inject messages, or register slash commands. The plugin is
`kind: standalone`, has no plugin dependencies and ships no `defaults.json`.
Hermes' native `register(ctx)` API registers the skill, not MCP servers. For typed
tools, **you** must add this entry to the selected profile's `config.yaml`
(normally `~/.hermes/config.yaml`), merging into any existing `mcp_servers` map:

```yaml
mcp_servers:
  rforge:
    command: node
    args: ["/absolute/path/to/installed/research-forge/adapters/mcp.mjs"]
```

Replace the path with the actual installed plugin location (inspect
`hermes plugins list`). Keep it absolute so calls from other workspaces work.
The launcher performs the same compatibility check and starts `rforge mcp serve`;
using the bare engine command here would skip the adapter's preflight. Run
`hermes mcp test rforge` and restart the session/gateway or use `/reload-mcp`.
The plugin does not add this entry, weaken approvals, or change any other config.
Remove this manual entry yourself when removing the plugin; after an update,
verify its path still points to the installed launcher.

If Hermes offers to install Node dependencies because it sees the Pi
`package.json`, decline: the native skill plugin needs none. MCP only needs
Node 22+ itself, with no npm dependencies.

```sh
hermes plugins update research-forge
# Removal:
hermes plugins disable research-forge
hermes plugins remove research-forge
```

Restart after upgrade or removal. No host approval, scheduling or runtime
configuration needs to be weakened for this plugin.

## Generic skill fallback

For other Agent Skills consumers, or hosts without native plugin support:

```sh
mkdir -p ~/.agents/skills/research-forge
curl -fsSL \
  https://raw.githubusercontent.com/TrebuchetDynamics/research-forge/main/skills/research-forge/SKILL.md \
  > ~/.agents/skills/research-forge/SKILL.md
```

Use the host's documented skill directory if it does not read `~/.agents/skills`.
Upgrade by repeating the download; remove by deleting only that skill copy.
Do not keep it alongside the native plugin in the same host. Then ask:
“Use the research-forge skill to research: `<question>`.”

## Recovery and boundaries

If the skill is missing, check the host's installed-plugin/package list, remove
old standalone copies, and reload or restart. If the engine check fails, install
or upgrade `rforge` separately; reinstalling the plugin will not repair a missing
binary. Plugin removal leaves `rforge`, research directories, provenance and
user-authored work intact. Remove the engine separately if desired.

## Maintainer packaging notes

The marketplaces point at the isolated `adapters/` plugin root because Claude
Code also scans a plugin's default `skills/` directory. Explicit paths alone
must not expose the repository's developer skills. The checked copy at
`adapters/skills/research-forge/SKILL.md` must match the canonical
`skills/research-forge/SKILL.md` byte-for-byte. After editing the canonical skill:

```sh
cp skills/research-forge/SKILL.md adapters/skills/research-forge/SKILL.md
go test ./internal/docs
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s adapters -p test_hermes.py
```

Claude lists the six public skill paths explicitly; Codex points at the isolated
`adapters/skills/` root. Pi lists the canonical methodology directory and the five
workflow directories explicitly, and its npm archive includes those workflow files.
Hermes registers only the canonical methodology `SKILL.md` directly. `plugin.yaml`
and workflow `agents/openai.yaml` policies use JSON syntax (a YAML subset) so Go's
offline conformance gate can parse them without adding a YAML dependency. The gate
is part of `make test`; it requires Node 22+ and checks manifests, versions, the
allowed skill set, explicit-only workflow policies, passive Pi command loading,
exact MCP registrations and prohibited runtime surfaces. The Pi adapter tests
build the real Go engine, check its current read/run tool set and CLI JSON parity,
and exercise missing/incompatible engines, preflight timeout and shutdown. Run:

```sh
node --test adapters/test
```

The npm archive must include both `adapters/pi.mjs` and `adapters/mcp.mjs`.
Keep the conformance allowlist aligned with reviewed MCP read/run additions
(such as `claim_check`), never approval/export operations.

Actual per-host install/upgrade/remove session qualification is live, opt-in
release work, not claimed by this offline check.

Host format references: [Claude manifest](https://code.claude.com/docs/en/plugins/manifest-reference),
[Claude marketplace](https://code.claude.com/docs/en/plugins/marketplace-reference),
[Codex packaging](https://developers.openai.com/plugins/build/plugins),
[Pi packages](https://pi.dev/docs/latest/packages),
[Pi native MCP](https://pi.dev/docs/latest/mcp),
[Pi extension MCP registration](https://pi.dev/docs/latest/extensions#mcp-servers),
[Hermes plugin API](https://hermes-agent.nousresearch.com/docs/developer-guide/plugins),
[Hermes MCP configuration](https://hermes-agent.nousresearch.com/docs/user-guide/features/mcp).
