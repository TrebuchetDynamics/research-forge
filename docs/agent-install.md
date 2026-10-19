# Agent installation

ResearchForge provides passive, skill-only plugins for Claude Code, Codex, Pi and
Hermes over the same standalone `rforge` engine. Install one integration per host.
Only the `research-forge` methodology skill is exposed; developer `*-tdd` skills
are not. There are no native tools, workflow commands, MCP servers, hooks,
schedules, background jobs, permission changes or runtime defaults in this release.
The skill remains model-invocable when research is relevant. Installation does
not start research or approve human review gates.

## Install the engine separately

Plugins do **not** download or install the binary. Install or upgrade it yourself:

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/research-forge/main/install.sh | bash
rforge version
rforge capabilities --json --min-rforge-version 0.1.22 \
  --require search=1 --require paper=1 --require brief=1 \
  --require compare=1 --require provenance=1
```

A released engine must meet the minimum version **and** required capabilities.
A `dev` engine is accepted only on verified capabilities, not its version label.
If the check fails (including an older engine without `capabilities`), upgrade
and rerun it before researching. With Go 1.26+, source installation is also
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
use the methodology skill. There is no Hermes-style slash command.

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

Requires Pi package support and Node 22 or later.

```sh
pi install git:github.com/TrebuchetDynamics/research-forge
# Or from a local checkout (keep the checkout in place):
# pi install .
```

Use `/reload` or start a new session, then
`/skill:research-forge <question>`. The extension entry point is intentionally a
no-op: no tools or commands are registered yet. Future native commands use the
`rforge_` prefix; the skill command keeps its portable name.

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
If Hermes offers to install Node dependencies because it sees the Pi
`package.json`, decline: this skill-only Hermes plugin needs none.

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

Pi uses an explicit single-skill directory; Hermes registers the canonical
`SKILL.md` directly. `plugin.yaml` uses JSON syntax (a YAML subset) so Go's offline
conformance gate can parse it without adding a YAML dependency. The gate is part
of `make test`; it checks manifests, versions, skill isolation and prohibited
runtime surfaces. Actual per-host install/upgrade/remove session qualification
is live, opt-in release work, not claimed by this offline check.

Host format references: [Claude manifest](https://code.claude.com/docs/en/plugins/manifest-reference),
[Claude marketplace](https://code.claude.com/docs/en/plugins/marketplace-reference),
[Codex packaging](https://developers.openai.com/plugins/build/plugins),
[Pi packages](https://pi.dev/docs/latest/packages).
