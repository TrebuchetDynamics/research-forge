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

- Real Python meta-analysis engine adapter after dependency/license review.
- Explicit unsupported-model handling beyond the fixture adapter.

Implemented:

- `CompareAnalysisEngines`, `BuildPyMAREFixtureResult`, and `rforge analysis engine-compare` produce PyMARE-style secondary meta-analysis engine comparison reports against metafor fixtures with environment locks/version capture, model-setting parity, warning capture, output deltas, and reviewer-visible disagreement reasons without changing the primary metafor analysis result.

## Recommended next slice

Add a real optional PyMARE adapter after dependency/license review, preserving the existing fixture-backed comparison contract and unsupported-model handling.
