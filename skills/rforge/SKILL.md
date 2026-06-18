---
name: rforge
description: Research any academic topic using rforge. Installs rforge automatically if not available. Covers literature search, systematic review, evidence extraction, and reproducible review packages.
---

# rforge — Academic Research Skill

Use this skill to search, collect, and synthesize academic literature using the `rforge` CLI. Works in any project — rforge is a standalone tool, not a dependency of your codebase.

## Step 0 — Check and install rforge

```sh
rforge --version
```

If the command is not found, install it:

```sh
go install github.com/TrebuchetDynamics/research-forge/cmd/rforge@latest
```

If Go is not installed, tell the user:

> rforge requires Go 1.22+. Install from https://go.dev/dl/ then run the above command.

## Step 1 — Create a project workspace

```sh
rforge project create <path> --title "<title>"
```

All subsequent commands use `--project <path>`. The project folder holds all outputs, sources, and the provenance log.

## Step 2 — Search and import papers

```sh
rforge search import \
  --source openalex \
  --query "<query>" \
  --pages 3 \
  --project <path>
```

Available sources: `openalex`, `arxiv`, `crossref`, `semantic-scholar`, `pubmed`, `europepmc`, `nasa-ads`, `doaj`, `core`

Run once per source to pull from multiple databases. Use `--from-year YYYY` to narrow the date range.

## Step 3 — Deduplicate

```sh
rforge duplicate report --project <path>
```

## Step 4 — Build a report

```sh
rforge report build --out <path>/report.md
rforge --project <path> ui          # open local web UI to browse results
```

## Guided systematic review workflow

For a full reproducible review package (question → search → screen → extract → analyze → package):

```sh
rforge forge init --project <path> --question "<research question>"
rforge forge status --project <path>
rforge forge next --project <path>
```

`rforge forge` walks through decision gates for protocol, sources, acquisition, parsing, screening, evidence extraction, analysis, and final packaging. Each gate requires human approval before proceeding.

To audit and replay a finished package:

```sh
rforge package audit <path>
rforge package replay <path>
```

## Provenance (always required)

Before finishing any run, ensure `<path>/provenance.json` exists and is populated:

```json
{
  "question": "<research question or task description>",
  "sources": ["openalex", "arxiv"],
  "queries": ["<exact query string used>"],
  "timestamp": "<ISO 8601>",
  "rforge_version": "<output of rforge --version>",
  "outputs": ["<relative paths to all saved files>"]
}
```

A run without `provenance.json` is incomplete. If `rforge` is not available, write this file manually.

## Rules

- Do not self-accept acquisition approval, privacy review, or LLM-suggestion queues. Surface them to the human and stop.
- Do not hit live scholarly APIs inside automated tests or CI.
- Do not package or distribute a review before privacy review is human-approved.
- A run without `provenance.json` is incomplete.
