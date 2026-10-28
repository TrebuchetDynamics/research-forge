# ResearchForge

**Search the literature. Keep the evidence trail.**

ResearchForge is an open-source command-line tool for finding papers across
scholarly sources, combining duplicate records, and turning search results into
briefs and bibliographies. It saves the queries, records, source failures and
checkpoints as ordinary files you can inspect, version and reuse.

Use it to scout a research question yourself or give a coding agent the same
workflow through a skill or MCP tools. For a formal review, a separate guided
project records the decisions that need human approval.

[Installation](./docs/install.md) · [Command reference](./docs/cli.md) ·
[Documentation](./docs/README.md) ·
[Releases](https://github.com/TrebuchetDynamics/research-forge/releases)

## Quick start

This walkthrough asks **what has been published about phononic thermal diodes?**
It searches two sources, saves a brief, and creates an unscreened bibliography.
It does not download papers or make inclusion decisions.

### 1. Install the current source

With Git and Go 1.26+ installed:

```sh
git clone https://github.com/TrebuchetDynamics/research-forge.git
cd research-forge
./install.sh --for cli --from-source
export PATH="$HOME/.local/bin:$PATH"
rforge version
```

This installs only the `rforge` binary for your user, without `sudo` or changes
to agent configuration. Building from the checkout keeps the executable and this
walkthrough on the same revision. See the [installation guide](./docs/install.md)
for release binaries, updates, Windows setup and agent integrations.

### 2. Search and inspect

The example writes to `my-research/`, separate from this repository's saved examples.

```sh
rforge search batch --out ./my-research/thermal-diodes \
  --query "phononic thermal diode" \
  --sources openalex,arxiv --continue-on-error --stats

rforge brief ./my-research/thermal-diodes \
  --out ./my-research/thermal-diodes/brief.md
```

Read `my-research/thermal-diodes/brief.md` for the search summary, paper list and
limitations. Check the reported failures: a completed command can still have
missing results from a rate-limited or unavailable source.

### 3. Build a reading list

```sh
rforge citations build --research-dir ./my-research --scope discovered
```

Open `my-research/CITATIONS.md`. It is built from saved metadata, without more
network calls or PDFs, and labeled **unscreened**: listing a paper does not mean
it was included in a review or read in full. Missing metadata stays missing.
The default bibliography mode, without `--scope discovered`, instead selects
papers with downloaded PDFs; see [bibliography scopes](./docs/cli.md#bibliography-scopes).

### What stays on disk

```text
my-research/
├── CITATIONS.md
└── thermal-diodes/
    ├── brief.md
    ├── manifest.json           # queries, sources and run summary
    ├── results.jsonl           # normalized source records
    ├── results-deduped.jsonl   # combined records
    ├── failures.jsonl          # source/query errors, including rate limits
    ├── batch-state.json        # resume checkpoint
    ├── search-stats.txt        # source coverage
    └── raw/                   # saved source/query responses
```

You can examine a [tracked search manifest](./research/open-source-project-search/manifest.json)
and its surrounding files to see the format without running a search.

## Scientific research examples

These saved investigations show the workflow on real physics questions:

- [Majorana zero modes — Comprehensive](./research/frontier-science/frontier-20261021-vfs2-live/majorana/report.md): distinguishing topological signatures from trivial alternatives in nanowires.
- [Nickelates — Standard](./research/frontier-science/frontier-20261021-vfs2-live/nickelates/report.md): comparing structural conditions and measurements relevant to pairing.

The reports retain source limitations and supporting artifacts. They are
agent-authored syntheses, not peer review or human scientific approval.
The [examples index](./research/frontier-science/README.md) links their citation
audits, provenance and reproduction instructions.

## Continue a search

Rerun the same search with `--resume` to skip completed query/source pairs and
continue within its saved budget:

```sh
rforge search batch --out ./my-research/thermal-diodes \
  --query "phononic thermal diode" \
  --sources openalex,arxiv --continue-on-error --stats --resume

rforge search stats --dir ./my-research/thermal-diodes
```

Resume does not replenish the request allowance. If the budget is exhausted,
you must explicitly raise its total cap to make more requests. Failed pairs stay
recorded until retried successfully. See [search budgets and stopping rules](./docs/cli.md#search-budgets-and-stop-reasons)
for deeper sweeps, query files, retry previews and refreshes.

## Resolve, rank, and synthesize

Beyond the walkthrough, you can:

- Resolve a paper by DOI, arXiv ID, URL or title with `rforge paper`.
- Prioritize a reading list with `rforge rank`.
- Compare overlap between search directories with `rforge compare`.
- Summarize several topic directories with `rforge lit`.
- Follow citation links with `rforge citations expand`.

These help organize discovery; a ranking or generated synthesis is not a
scientific verdict. Find arguments and examples in the [command reference](./docs/cli.md).

## When a search becomes a review

Use a guided project when you need attributable screening, extraction, analysis
and export decisions—not just a folder of search results:

```sh
rforge project create ./my-review --title "Phononic thermal diodes"
rforge forge init --project ./my-review \
  --question "Which mechanisms and validation conditions govern thermal rectification?"
rforge forge status --project ./my-review
```

The workflow stops at required human approvals. It does not approve its own
question, inclusion decisions, full-text acquisition, evidence, analysis, claims
or package export. See the [review-package guide](./docs/reproducible-review-package.md)
for the complete workflow and offline package audit/replay commands.

## Use it from an agent

Native integrations are available for **Claude Code, Codex, Pi and Hermes**.
They expose research skills and engine tools without changing host approvals,
permissions or schedules. The CLI-only install above does not install them.

Follow [agent setup](./docs/agent-install.md) for your host, then ask:

> Use research-forge to research phononic thermal diodes. Save the search,
> source coverage and a report with references and limitations.

The [research skill](./skills/research-forge/SKILL.md) defines the workflow and
human gates. Other stdio MCP clients can connect directly to `rforge mcp serve`;
see [manual MCP setup](./docs/cli.md#mcp-stdio-server).

## Trust boundaries

- **Sources can fail.** API quotas, outages and incomplete metadata affect
  coverage. Inspect failures and source statistics; retrieval is not exhaustive.
- **Local storage is not offline search.** Queries go to the selected source
  APIs; results are saved locally. See [outbound data flow](./docs/outbound-api-data-flow.md).
- **Metadata is not evidence of a result.** Briefs, rankings and unscreened
  bibliographies do not replace reading, screening or scientific judgment.
- **Acquisition and sharing need review.** Full-text retrieval is limited to
  supported legal open-access routes. Human privacy, licensing and export gates
  still apply; paywall bypass is not supported.

## Development

ResearchForge uses a Go core, SQLite and an optional Go + HTMX local web UI.
Start with the [contributing guide](./CONTRIBUTING.md),
[developer setup](./docs/developer-setup.md) and
[architecture tour](./docs/architecture.md). The full check is `make check`;
normal tests are offline and deterministic.

[Product requirements](./RESEARCH-FORGE-PRD.md) · [Roadmap](./ROADMAP.md) ·
[Development skills](./SKILLS.md)

<details>
<summary>Decision-gated scope and maintenance</summary>

The MIT License decision is tracked in issue #1 and
[owner decisions](./docs/owner-decisions.md). The local Go + HTMX UI decision is
tracked in issue #2 and ADR 0006. These are recorded decisions, not a request for
new approval. Maintainers can inspect the checklist with `make todo-audit`,
`make todo-completion-audit` and `make decisions-markdown`.

</details>

## License

[MIT License](./LICENSE) (SPDX: `MIT`) © 2026 Trebuchet Dynamics.
