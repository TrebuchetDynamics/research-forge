# Variance floor for scientific benchmarking meta-analysis

status: accepted

Scientific benchmarking meta-analysis pools a single continuous outcome per study (e.g., solar-to-hydrogen efficiency %) using `rma(yi, vi)` in R/metafor. Materials science device papers routinely report only a peak efficiency value with no standard error or confidence interval — requiring fully reported uncertainty would discard the majority of retrievable studies and make the analysis unrunnable on the existing corpus. We decided to accept studies without reported variance and impute a **user-specified floor variance** at `rforge analysis prepare` time via `--variance-floor <v>` (default `0.0025`, corresponding to ±5% relative uncertainty, a typical instrument precision floor in photoelectrochemistry). The floor value is stored as a first-class parameter in the analysis manifest and provenance record. Studies that carry imputed variance are tagged `vi_source: floor` in the input table. A sensitivity run excluding floor-imputed studies is emitted automatically alongside the main run.

## Considered options

- **Require reported uncertainty** — statistically cleanest, but excludes most materials science papers and makes the analysis unrunnable on the current corpus.
- **Fixed coefficient of variation** — simpler (σ = k·yi), but conflates absolute instrument precision with relative measurement error and obscures the assumption.
- **Large sentinel vi** — down-weights studies without uncertainty by assigning an inflated variance. Rejected because it implicitly treats absence of reporting as evidence of unreliability, which is a domain assumption we don't want baked in silently.

## Consequences

- The `--variance-floor` value must be documented in every report and reproducible package as a sensitivity parameter; reviewers should treat floor-imputed studies with appropriate caution.
- The automatic sensitivity run is not optional — it is part of the analysis output contract. Analysis runs without the sensitivity artifact are considered incomplete by `rforge package audit`.
- A new `raw-continuous` effect measure is added to `rforge analysis prepare --effect`, distinct from the existing arm-pair calculators (`smd`, `log-odds-ratio`, etc.).
- Downstream: when a paper reports a CI or SE, those values take precedence over the floor; extraction schemas for benchmarking outcomes must include optional `ci_lower`, `ci_upper`, and `se` fields.
