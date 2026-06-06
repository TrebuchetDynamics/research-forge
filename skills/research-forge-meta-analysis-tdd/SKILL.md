---
name: research-forge-meta-analysis-tdd
description: Build ResearchForge meta-analysis and statistical reporting with strict TDD. Use for effect sizes, R metafor adapter, analysis inputs, forest/funnel plots, heterogeneity, sensitivity analysis, or reproducible statistical notebooks.
---

# ResearchForge Meta-Analysis TDD

Use this skill for Milestone 6 statistical analysis work.

## Quick start

1. Read `DEVELOPMENT_PLAN.md` Milestone 6 and PRD sections 4.5 and 7.7.
2. Choose one statistical behavior with known fixture data.
3. Write a failing test against expected inputs, scripts, or outputs.
4. Implement minimally, preferring auditable generated scripts.
5. Refactor adapter boundaries while preserving reproducibility.

## TDD contract

- **Red:** failing test for effect-size calculation, analysis input generation, R script generation, result parsing, or audit metadata.
- **Green:** minimal implementation; use deterministic fixtures.
- **Refactor:** separate statistical model selection from external engine execution.
- **Receipt:** run targeted tests; external R/metafor tests must be skipped or opt-in when unavailable.

## Slice order

1. Analysis input table from accepted evidence.
2. Effect-size helper interface and first calculator.
3. R/metafor script generation.
4. Engine version capture.
5. Result file manifest.
6. Forest plot artifact registration.
7. Funnel plot artifact registration.
8. Heterogeneity metric parsing.
9. Sensitivity-analysis scaffold.
10. Fyne analysis view model.

## Verification gate

Done requires:

- fixture calculations match known expected values or generated scripts are golden-tested;
- analysis inputs are snapshot and trace back to evidence IDs;
- external engine version and parameters are recorded;
- missing R/metafor produces a clear actionable error.

## Red lines

- Do not invent statistical interpretations in code or reports.
- Do not run opaque analysis without saving inputs/scripts/outputs.
- Do not make R mandatory for commands unrelated to analysis.

## References

- [Statistical reproducibility](references/statistical-reproducibility.md)
