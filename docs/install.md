# Installation

`install.sh` is the single installation entry point. It installs ResearchForge
for the **current user**: the `rforge` engine, the public agent skills, and the
native Claude Code, Codex, Pi and Hermes integrations. Nothing is installed
system-wide, no step needs `sudo`, and no unrelated host configuration is
changed.

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/research-forge/main/install.sh | bash
```

From a checkout, `./install.sh` does the same and prefers the local skill copies:

```sh
git clone https://github.com/TrebuchetDynamics/research-forge
cd research-forge
./install.sh
```

Run `./install.sh --help` for every option, or `./install.sh --list` to see what
is available on this machine.

## What gets installed

| Component | Key | Installed to | Notes |
| --- | --- | --- | --- |
| Engine | `binary` | `~/.local/bin/rforge` | Release binary by default; verified against the release checksums |
| Public skills | `skills` | `~/.agents/skills/<name>/SKILL.md` | Generic Agent Skills scope: `research-forge`, `lit`, `paper`, `brief`, `compare`, `replicate` |
| Claude Code | `claude` | Claude's user plugin scope | `claude plugin marketplace add` + `claude plugin install` |
| Codex | `codex` | Codex's user plugin scope | `codex plugin marketplace add` + `codex plugin add` |
| Pi | `pi` | Pi user packages | `pi install git:github.com/TrebuchetDynamics/research-forge` |
| Hermes | `hermes` | Current Hermes profile | `hermes plugins install` + `hermes plugins enable` |

Only the six public research skills are installed. The repository's developer
`*-tdd` skills are never published to user scope.

After installing a host integration, open a new host session so it discovers the
skills. Available components are installed by default; a host whose CLI is absent
is reported as skipped and does not fail the run.

## Choose by intent

Most people want one of three things. A profile selects the components for that
intent, so you do not have to know the component names first:

| You want | Command | Installs |
| --- | --- | --- |
| `rforge` commands in your shell | `./install.sh --for cli` | the engine only |
| the research skills in your agent | `./install.sh --for skills` | public skills plus each installed host's plugin |
| typed MCP tools in your agent | `./install.sh --for mcp` | the engine plus host plugin/MCP registration |
| everything | `./install.sh --for all` | both (this is the default) |

Native plugins deliver skills and MCP registration together, so `skills` and
`mcp` differ mainly in whether the engine is installed: the skill works without
it, MCP tool calls need it. `--for mcp` also leaves the generic `~/.agents/skills`
copy alone because the plugins already provide those skills. With no host CLI
installed, `--for mcp` installs the engine for a manual stdio client entry; see
[MCP setup](./cli.md#mcp-stdio-server).

A profile is a convenience selection, not a per-component demand: a host whose
CLI is absent is skipped rather than failing the run, exactly like the default
suite. Combine a profile with `--skip` to drop a host, and use `--only` instead
when you want to name components yourself.

```sh
./install.sh --for mcp --skip codex
./install.sh --for cli --dry-run
```

## Choose components

```sh
./install.sh --list                        # components and availability
./install.sh --menu                        # interactive selection (needs a terminal)
./install.sh --only binary,skills          # exactly these
./install.sh --only binary,skills,hermes   # engine, skills, Hermes plugin
./install.sh --skip codex                  # everything else
./install.sh --all                         # every available component (default)
```

`--menu` lists the components with a checkbox; toggle them by number, `a` selects
all, `n` none, and Enter installs. Without a terminal, use `--only` or `--skip`.

## Engine options

The previous binary installer options keep working:

```sh
./install.sh --only binary                          # latest release
./install.sh --only binary --version v0.1.21        # a specific release
./install.sh --only binary --dir ~/bin              # a different directory
./install.sh --only binary --from-source            # build with Go 1.26 instead
```

`RFORGE_VERSION`, `RFORGE_INSTALL_DIR` and `RFORGE_INSTALL_FROM_SOURCE` remain
supported. For an internal mirror, set `RFORGE_RELEASE_BASE` (release downloads)
and `RFORGE_RAW_BASE` (skill downloads). A source build installs into the same
directory with `GOBIN`.

The engine is verified after installation: the installer runs `rforge version`,
then `rforge capabilities --json --min-rforge-version <x> --require …` for the
search, paper, brief, compare, provenance and mcp contracts. A download whose
sha256 does not match `checksums.txt` is never installed. If verification fails,
the component fails (non-zero exit) while the other components still run —
except for one recoverable case: when the downloaded release fails the gate and
the installer runs from a buildable checkout with Go, it rebuilds that checkout
(reporting the fallback) rather than leaving a broken engine behind.

### PATH

The default directory is `~/.local/bin`, which needs no privileges. If it is not
on your `PATH`, the installer prints the exact line to add; it never edits your
shell profile itself:

```sh
export PATH="$PATH:$HOME/.local/bin"
```

If another `rforge` resolves earlier on `PATH`, the installer says so, so a stale
copy cannot silently shadow the new one.

## Update

```sh
./install.sh update
```

`update` upgrades only the components detected as already installed, and reports
the rest as `not installed`. Each host uses its own supported upgrade command:

| Component | Update command |
| --- | --- |
| `binary` | reinstall the current release (checksum-verified) |
| `skills` | copy the current public skills, keeping a backup of replaced copies |
| `claude` | `claude plugin marketplace update` + `claude plugin update` |
| `codex` | `codex plugin marketplace upgrade` + re-add the plugin |
| `pi` | `pi update git:github.com/TrebuchetDynamics/research-forge` |
| `hermes` | `hermes plugins update research-forge` |

Reinstalling without `update` is also safe: it is idempotent and reports
`current` for unchanged skill files.

## Dry run

```sh
./install.sh --dry-run
./install.sh --only binary,hermes --dry-run
```

`--dry-run` prints every command it would run and changes nothing: no files, no
host plugin state, no configuration. Detection commands (`claude plugin list`,
`pi list`, `hermes profile list`, …) are read-only and do still run so the plan is
accurate.

## Hermes profiles

The installer uses the self-contained [`hermes-plugin/`](../hermes-plugin/)
subdirectory, not the full repository tree, so Hermes scans only the passive
entry point, manifest and methodology skill. The equivalent manual install is:

```sh
hermes plugins install https://github.com/TrebuchetDynamics/research-forge/tree/main/hermes-plugin --no-deps
hermes plugins enable research-forge
```

Existing repository-root installs keep their original tree; reinstall using the
subdirectory URL to switch. No Hermes profile is changed merely by this packaging
change.

By default the **current** Hermes profile is used (`HERMES_PROFILE` if set, else
Hermes' sticky default). Name profiles explicitly to target others:

```sh
./install.sh --only hermes --hermes-profile research-forge
./install.sh --only hermes --hermes-profiles research-forge,default
```

An unknown profile fails before anything is installed and lists the profiles
detected from `hermes profile list`. Profiles that were not named are never
modified. The installer does **not** register MCP servers or edit `config.yaml`;
that entry stays manual and reversible, as described in the
[agent installation guide](./agent-install.md#hermes).

## Verify

```sh
rforge version                  # runs from any directory
rforge doctor                   # local environment checks
rforge capabilities --json      # build and command contracts
claude plugin list              # contains research-forge
codex plugin list               # contains research-forge
pi list                         # contains the research-forge package
hermes plugins list             # contains research-forge, enabled
ls ~/.agents/skills             # the six public skills
```

## Avoid duplicate skills

Installing the generic `skills` component next to the **Pi package** would make Pi
discover every skill twice, because Pi reads `~/.agents/skills` as well as its
packages. The installer detects that and reports `skills` as skipped, pointing at
the package as the single source.

A standalone copy left over from an earlier curl install can shadow a plugin in
the same host. When the installer installs a native plugin, it detects such
legacy copies (`~/.claude/skills/research-forge`, `~/.codex/skills/…`,
`~/.pi/agent/skills/…`, `~/.hermes/skills/…`, and per-profile Hermes skill
directories) and moves them to `<name>.bak` instead of deleting them. Existing
copies are never overwritten without a backup.

## Removal

Remove only what you no longer want; research directories, provenance and
user-authored work are untouched.

```sh
rm ~/.local/bin/rforge                    # engine
rm -rf ~/.agents/skills/{research-forge,lit,paper,brief,compare,replicate}

claude plugin uninstall research-forge@research-forge
claude plugin marketplace remove research-forge

codex plugin remove research-forge@research-forge
codex plugin marketplace remove research-forge

pi remove git:github.com/TrebuchetDynamics/research-forge

hermes plugins disable research-forge
hermes plugins remove research-forge
# and remove any manual mcp_servers.rforge entry from your Hermes config
```

If you used `--hermes-profile`, add the same `--profile` flag to the Hermes
commands.

## Troubleshooting

| Symptom | Cause and fix |
| --- | --- |
| `binary` fails: `installed engine … does not meet the required rforge X` | The release predates the capabilities the plugins need. From a buildable checkout with Go, the installer rebuilds the checkout automatically and says so; otherwise install a release at or above that version, or run `make install`. |
| `binary` fails: `checksum mismatch` | The download was corrupted or the release is inconsistent; rerun, then report it if it persists. Nothing was installed. |
| `--dry-run` reports a host `not found` | That host's CLI is not on `PATH`; install it or select other components with `--only`. |
| A host component fails while others install | Components are independent; rerun the failed one with `--only`, or `./install.sh update`. |
| Skills appear twice in a host | A legacy standalone copy is still present; the installer moves it to `.bak` when it installs the plugin. Remove it manually if you installed the plugin earlier. |
| `rforge` not found after install | `~/.local/bin` is not on `PATH`; add the `export PATH=…` line printed by the installer. |

## Development

The installer is bash 3.2-compatible (macOS default bash) and has no runtime
dependency beyond the tools already used to install a release (`curl` or `wget`,
plus `sha256sum`/`shasum`/`openssl` for verification, which are optional).

Its end-to-end tests run offline against mocked host CLIs in an isolated `HOME`:

```sh
go test ./internal/installer
make install-script-smoke
```

`make check` runs both as part of the normal gate.
