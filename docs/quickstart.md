# Quickstart

Prerequisites: [install the CLI](install.md#choose-by-intent) and ensure `rforge`
is on your `PATH`. Start in a writable directory with no existing `demo` project,
and run each step from that same directory.

## Create a project

```sh
rforge project create demo --title "Artificial Photosynthesis Review"
```

This creates the `demo` workspace and its manifest, lockfile and provenance.

## Search and import metadata (live)

This step requires network access to OpenAlex. It is a live source query, not an
offline fixture: results and source availability can change. The command imports
records directly into the project's library; no intermediate input file is needed.

```sh
rforge --project demo search import --source openalex --query "artificial photosynthesis" --pages 1 --limit 20
```

## Inspect and prepare the review

After the import succeeds, use the same project for duplicate inspection,
screening configuration and an initial report scaffold:

```sh
rforge --project demo duplicate report
rforge --project demo screen configure --reason "wrong population"
rforge --project demo report build --out demo/report.md
```

`duplicate report` lists possible duplicates; it does not merge them. Configuring
exclusion reasons does not make screening decisions. `report build` writes
`demo/report.md` relative to your current directory, but this scaffold is not a
completed or validated review. Screening, source-backed evidence extraction and
analysis still require separate work; see the [CLI reference](cli.md).

The repository's [fixture tests](fixture-policy.md) validate commands offline;
this live tutorial is not an offline test or proof of research completeness.
