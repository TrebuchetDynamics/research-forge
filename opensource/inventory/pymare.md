# PyMARE study note

- Repository/ecosystem: `neurostuff/PyMARE`.
- Area: Python meta-analysis and meta-regression engine.
- Disposition: `adapter-only`.
- License/action constraint: call only as an optional external Python engine after dependency/license review; record environment and package versions; do not vendor code.

## Why it matters

ResearchForge currently centers R/metafor for statistical correctness. PyMARE is a useful secondary engine for cross-checks, Python-centric environments, and method comparison reports.

## Patterns to learn

- Analysis engines should be swappable behind explicit model-setting contracts.
- Engine comparison is valuable only when inputs, settings, versions, and warnings are identical and recorded.
- Disagreements between engines should be visible in reports rather than hidden.
- Python dependency environments need reproducibility metadata similar to R/metafor scripts.

## ResearchForge status

Implemented nearby capabilities:

- Analysis input snapshots from accepted evidence.
- R/metafor script generation and opt-in real integration tests.
- Sensitivity, subgroup, meta-regression, publication-bias, and Bayesian-normal scaffolds.
- Analysis artifacts with checksums and version capture.

Missing features:

- Python meta-analysis engine adapter.
- Cross-engine comparison report against the same input snapshot.
- Python environment lock/version capture.
- Explicit handling when engines disagree or do not support a selected model.

## Recommended next slice

Add a secondary-engine comparison report that can run fake-backed PyMARE adapter tests against existing metafor fixtures and show output deltas without changing the primary analysis result.
