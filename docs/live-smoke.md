# Live smoke tests

Normal ResearchForge validation is network-free. Live source checks are opt-in and use the same connector code paths as the CLI.

## All scholarly sources

```sh
make source-live-smoke
```

This sets `RFORGE_RUN_LIVE_SOURCE_SMOKE=1` and runs `TestOptInLiveSourceConnectorSmoke` against OpenAlex, arXiv, Crossref, Semantic Scholar, Europe PMC, PubMed, and Unpaywall when its email is configured.

## Semantic Scholar

```sh
make semantic-scholar-live-smoke
```

Optional configuration:

- `RFORGE_SEMANTIC_SCHOLAR_API_KEY` — optional API key sent as `x-api-key`.
- `RFORGE_SEMANTIC_SCHOLAR_URL` — endpoint override.
- `RFORGE_SEMANTIC_SCHOLAR_MAX_RETRIES` — CLI retry count for quota/transient failures.

## Biomedical connectors

```sh
make biomedical-live-smoke
```

This runs only the PubMed and Europe PMC subtests.

Optional biomedical configuration:

- `RFORGE_PUBMED_API_KEY` — optional NCBI API key.
- `RFORGE_PUBMED_TOOL` — optional NCBI `tool` value.
- `RFORGE_PUBMED_EMAIL` — optional NCBI contact email.
- `RFORGE_PUBMED_URL` — PubMed/E-utilities endpoint override.
- `RFORGE_EUROPEPMC_URL` — Europe PMC endpoint override.

The live smoke test checks that each connector returns at least one titled record for a lightweight query. PubMed raw provenance redacts API keys, and normal test runs never require network access.
