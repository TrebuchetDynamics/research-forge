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
- Leave-one-out sensitivity analysis through `rforge analysis sensitivity <run-id> --method leave-one-out`.
- Binary-outcome log odds ratio and risk ratio effect-size preparation through `rforge analysis prepare <run-id> --effect log-odds-ratio|risk-ratio`.
- Egger-style publication-bias diagnostic through `rforge analysis publication-bias <run-id> --method egger`.
- Weighted numeric moderator meta-regression through `rforge analysis meta-regression <run-id> --moderator <name> --value <paper>=<number>`.
- Categorical subgroup pooled estimates through `rforge analysis subgroup <run-id> --variable <name> --group <paper>=<group>`.
- `RunMetafor` writes reproducible SVG forest/funnel plot artifacts with checksums alongside scripts and captured output.
- `AnalysisArtifactManifest` / `rforge analysis run` bundle forest/funnel SVGs, plot settings, checksums, R/metafor script metadata, engine versions, warnings, and report embedding metadata in `analysis/<run-id>-artifact-manifest.json`.
- The forge workflow DAG includes analyze/report checkpoints with inputs, outputs, provenance actions, and restart-safe skips.
- Bayesian normal-normal approximation alternative through `rforge analysis bayesian <run-id> --method normal-approx`.

Missing features:

- Additional effect-size calculators beyond standardized mean difference, log odds ratio, and risk ratio.
- Richer subgroup analysis and meta-regression command UX beyond direct CLI values.
- Influence diagnostics beyond leave-one-out estimates.
- Richer publication-bias tests beyond Egger-style regression.
- Higher-quality publication-ready forest/funnel styling beyond current reproducible SVG diagnostics.
- Rich Bayesian alternatives as separate engines beyond normal-approximation scaffold.

## Recommended slice

Add influence diagnostics and richer sensitivity artifacts on top of the implemented leave-one-out command.

Acceptance target:

```sh
rforge analysis sensitivity <run-id> --method influence
```
