# metafor study note

- Repository/ecosystem: R `metafor` package.
- Area: meta-analysis, heterogeneity, forest/funnel plots, meta-regression.
- Disposition: `adapter-only`.
- License/action constraint: call installed R/metafor through explicit external-tool adapter; do not reimplement from package internals.

## Why it matters

`metafor` is a gold-standard statistical package for meta-analysis. ResearchForge should prefer auditable established statistics over generated summaries.

## Patterns to learn

- Keep analysis inputs, scripts, model settings, warnings, and outputs reproducible.
- Record engine/version information in lockfiles.
- Expose heterogeneity and sensitivity diagnostics, not only a pooled estimate.
- Treat plot artifacts as reproducible outputs with checksums.

## ResearchForge status

Implemented nearby capabilities:

- Analysis preparation from accepted evidence.
- R/metafor script generation.
- Safe external-command wrapper.
- Opt-in R/metafor integration tests.
- Analysis export.

Missing features:

- Richer effect-size calculators.
- Subgroup analysis and meta-regression command UX.
- Leave-one-out and influence diagnostics.
- Publication-bias tests.
- Higher-quality forest/funnel plot artifacts.
- Bayesian alternatives as separate engines.

## Recommended slice

Add a sensitivity-analysis command that consumes an existing analysis run and produces leave-one-out/influence artifacts.

Acceptance target:

```sh
rforge analysis sensitivity <run-id> --method leave-one-out
```
