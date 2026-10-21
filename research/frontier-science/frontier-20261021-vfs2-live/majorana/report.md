# Majorana nanowires: evidence that discriminates, and evidence that does not

**Exploratory, not gold — agent_reported scientific draft.** Research question: What evidence is sufficient to establish topological Majorana zero modes in semiconductor–superconductor hybrid nanowires, and which signatures also arise from trivial Andreev bound states?

<a id="findings"></a>
## Concise findings

A local zero-bias peak, even near **2e²/h**, is not an adequate standalone diagnostic in this inspected evidence: experimental Andreev peaks and model disorder peaks are explicit alternatives [9,10]. Combined local/nonlocal measurements make a stronger, but model- and analysis-dependent case [3,5,6]. **This dossier does not establish topological Majorana modes, independent replication of a complete topological diagnostic, or non-Abelian braiding.** The positive protocol study itself describes its result as a prerequisite for fusion/braiding, not a braiding demonstration [5]. This is a source-limited conclusion, not a claim that topology is impossible or that no later experiment could establish it.

<a id="method"></a>
## Scope and search method

Metadata retrieval cutoff: **2026-10-21**, with primary abstract checks that same day. ResearchForge performed the genuine retrieval/dedupe, paper resolution, citation expansion and mechanical summaries; the Pi host wrote this interpretation. [Queries](queries.txt), [manifest](manifest.json), [checkpoint](batch-state.json), [current coverage](coverage-stats.log) and [commands](commands.jsonl) preserve exact inputs and outputs.

Selected depth **Comprehensive**; effective caps **30 query variants, 20 sources, 100 connector Search calls**, limit **20** per query/source. The selected plan has **20 queries and four sources** (OpenAlex, Crossref, Semantic Scholar, arXiv). Final cumulative **100/100** calls, **599 unique deduped records**, stop **budget_exhausted**; consecutive low-yield count **0**, against saturation requirement **5** with threshold **1**. Coverage lists **19** reached query variants; the final review/limitations variant is not represented there. This is capped, unsaturated retrieval, not exhaustive coverage. Earlier initial/resume processes each timed out at **120 seconds**; the explicitly owner-authorized second resume finished normally. These process interruptions are not physical evidence or saturation. Final manifest records **10 failed pairs**; see [failures](failures.jsonl), predominantly Semantic Scholar rate limits. Earlier failures and overwritten historical-output checksums remain visible in immutable historical receipts; only the current applicable search receipt is indexed. The receipt labels its generic depth `standard`, whereas actual argv/checkpoint/manifest say `comprehensive`; this discrepancy is disclosed, not repaired in campaign data.

The run used **3/5** allowed paper resolutions and **1/3** allowed citation seeds, depth **1**, maximum **30 records / 6 API calls / 0 retries** per seed. No additional searches were performed during synthesis. HTTP attempts (including internal retries) are unknown, not equated to connector calls. These are operational counts; see [reproduction](reproduce.md) and [run state](../run.json).

<a id="evidence"></a>
## Evidence and signature comparison

The table reports **what each inspected source says**, not host-certified measurements. All conditions, original source versions and reasons for unavailable values are in [evidence.csv](evidence.csv); a title alone was never used as a measurement. Each source row has a short attributable passage in [primary-checks.json](primary-checks.json).

| Source / signature | Observation or calculation | Rival interpretation / evidentiary limit |
|---|---|---|
| <a id="m1"></a>M1 [1] historical quantized plateau | Original abstract claimed a **2e²/h** plateau in InSb–Al nanowires. | **Retracted**: not surviving evidence for quantized Majorana conductance; paired with M2, never counted as confirmation. |
| <a id="m2"></a>M2 [2] primary retraction | Authors explicitly say they can no longer claim quantized Majorana conductance after re-analysis/recalibration. | Withdrawal does not establish which trivial mechanism caused every feature. |
| <a id="m3"></a>M3 [3] local/nonlocal/thermal theory | Models pristine, dot/inhomogeneous and disordered finite wires; local peaks fail to distinguish them, nonlocal signals can be weak. | Advocates combined electrical/thermal diagnostics; a conditional theoretical guide, not an experimental sufficiency theorem. |
| <a id="m4"></a>M4 [4] full-shell junction study | Short junctions lack subgap states while longer junctions show them; authors explain robust zero-bias peaks using junction Andreev levels. | Full-shell geometry and robust peaks alone do not remove junction-dot alternatives. Material species and numerical field/T ranges are unavailable in the inspected abstract. |
| <a id="m5"></a>M5 [5] positive topological-gap-protocol study | Gated InAs–Al quasi-1D wires defined in a 2D electron gas; reports bulk-gap closing/reopening and simultaneous end peaks; extracted maximum gaps **20–60 μeV**. | Inference of high topological probability is conditioned on disorder simulations/protocol. Not direct fusion/braiding proof; M6 disputes the analysis. |
| <a id="m6"></a>M6 [6] adverse protocol comment | Legg argues that gap/topology definitions and field/bias windows, resolution and cutter-pair counts affect whether the protocol passes. | **Non-peer-reviewed preprint** criticism in this audit; host did not rerun its analysis or exhaustively locate responses. Not proof all devices are trivial. |
| <a id="m7"></a>M7 [7] correlated subgap experiment | At **zero field**, end-to-end correlations in a **300 nm** device are reported reduced by factor **four** in a **900 nm** device. | These are subgap-feature coincidences, not equivalent to a verified topological zero-mode pair or a matched replication of M5. |
| <a id="m8"></a>M8 [8] end-correlated zero-bias model | Dot-at-one-end model predicts identical end zero-bias peaks for Majorana states but not its trivial Andreev states. | Model-specific discriminator. Cannot extend it to every disorder configuration or equate all subgap correlations with topology. |
| <a id="m9"></a>M9 [9] disorder alternative | Calculations produce trivial sharp Andreev peaks near **2e²/h**, interpreting existing Delft/Yu datasets; field/barrier stability is discussed. | Theory/reinterpretation, not independent experimental replication or a host-established cause of the original data. |
| <a id="m10"></a>M10 [10] weakly dissipative experiment | InAs–Al Andreev peaks near **2e²/h**; with increasing temperature, peak growth or split-peak merging is reported. | Environmental Coulomb blockade competes with thermal broadening. Height and temperature behavior require environmental controls, not a unique Majorana inference. |

<a id="rivals"></a>
## Competing hypotheses, contradictions and replication

The topological interpretation of M5 is stronger than a lone local peak because it combines bulk and end signatures, but remains an interpretation tied to simulations and protocol definitions [5]. The alternatives are not interchangeable: junction-confined Andreev levels are directly investigated in M4; disorder-induced peaks in M9 are calculations and dataset reinterpretation; environmental effects in M10 change Andreev temperature dependence [4,9,10]. M6 challenges diagnostic robustness, not directly the existence of every possible topological phase [6]. M3 additionally cautions that weak nonlocal conductance can limit the proposed bulk-gap diagnostic in disordered systems [3]. We cannot resolve this dispute from abstracts.

End correlations deserve particular care: M8 predicts **zero-bias** correlations within its dot-wire model, while M7 observes **subgap** correlations at zero field [7,8]. They are neither the same test nor a logical contradiction. Generic correlation is not sufficient topological proof. M3/M8 share Sau/Das Sarma authors, and M9 overlaps Pan/Das Sarma: these are not independent theoretical or experimental confirmations. The retraction is the original authors’ correction record, not independent replication. None of the inspected primary abstracts establishes independently reproduced, matched complete-protocol results on separate groups’ devices. That last statement is an audit gap, not a literature-wide nonexistence claim [2,3,5,7–9].

<a id="predictions"></a>
## Five falsifiable proposals

These are conditional **host proposals**, not five established properties or a universal acceptance test. Full refuters, rationale and primary locators are in [predictions.json](predictions.json).

1. **End coincidence:** in the M8 dot-wire regime, candidate topological peaks recur at both ends over the same tuning window. A persistent one-ended peak with verified two-ended sensitivity refutes that assignment in that regime [8]; generic coincidences still need alternatives tested [3,7].
2. **Analysis stability:** a fixed, preregistered M5-style joint gap/end diagnostic survives declared window/resolution sensitivity checks. Loss of the pass under reasonable fixed-data checks refutes protocol robustness, not every possible topological state [5,6].
3. **Junction control:** M4's junction-dot explanation predicts features track junction configuration. A matched, sensitivity-controlled experiment invariant to junction configuration refutes that explanation in the tested regime [4].
4. **Disorder stability:** M9's disorder-peak model predicts limited field/barrier stability. Calibration-controlled stability inconsistent with that tested model across independent devices refutes that alternative only [9]; the retraction motivates calibration controls [2].
5. **Environmental temperature response:** in M10's verified weakly dissipative Andreev regime, peak growth or split-peak merging should occur at selected settings. Exclusively conventional thermal suppression across comparable settings refutes that conditional prediction [10].

<a id="limits"></a>
## Limitations and evidence-proportionate conclusion

This is an abstract-led selected-source synthesis, not a screened systematic review. Numerical field and temperature windows, uncertainties, device statistics and independent device/dataset matching are mostly unavailable; unknowns remain explicit rather than guessed. No full texts, PDFs, raw experimental datasets, braiding data, or peer review were acquired/performed. Version histories and the explicit primary retraction were checked; a complete correction/retraction registry and all subsequent responses were not. A linked journal DOI does not prove that the inspected preprint abstract is the definitive journal text. See [citation audit](citation-audit.json).

**Conclusion:** the retrieved evidence supports demanding convergent bulk/end diagnostics with prespecified analysis, calibration, environmental/disorder controls and independent replication, rather than certifying topology from one peak [2–10]. It does **not** supply a universally sufficient experimentally verified criterion or independently validate non-Abelian operations [3,5,6]. Observations (transport features), model-dependent interpretation (topology/trivial mechanisms), and unresolved proof/replication are deliberately separate. Mechanical provenance validates saved-file integrity and run linkage, never scientific correctness or human approval.

## Numbered references

1. **M1** — Zhang, Hao et al. *Quantized Majorana conductance*. Record date 2017/10/29; retracted (inspected record). DOI: 10.1038/nature26142. [Primary record](https://arxiv.org/abs/1710.10701). Submitted on 29 Oct 2017]
2. **M2** — Zhang, Hao et al. *Retraction Note: Quantized Majorana conductance*. Record date 2021/03; published (inspected record). DOI: 10.1038/s41586-021-03373-x. [Primary record](https://www.nature.com/articles/s41586-021-03373-x).
3. **M3** — Pan, Haining et al. *Three-terminal nonlocal conductance in Majorana nanowires: Distinguishing topological and trivial in realistic systems with disorder and inhomogeneous potential*. Record date 2020/09/24; preprint (inspected record). DOI: 10.1103/PhysRevB.103.014513. [Primary record](https://arxiv.org/abs/2009.11809v3). Submitted on 24 Sep 2020 ( v1 ), last revised 24 Jan 2021 (this version, v3)]
4. **M4** — Valentini, Marco et al. *Nontopological zero-bias peaks in full-shell nanowires induced by flux tunable Andreev states*. Record date 2020/08/05; preprint (inspected record). DOI: 10.1126/science.abf1513. [Primary record](https://arxiv.org/abs/2008.02348v3). Submitted on 5 Aug 2020 ( v1 ), last revised 10 Aug 2021 (this version, v3)]
5. **M5** — Aghaee, Morteza et al. *InAs-Al Hybrid Devices Passing the Topological Gap Protocol*. Record date 2022/07/06; preprint (inspected record). DOI: 10.1103/PhysRevB.107.245423. [Primary record](https://arxiv.org/abs/2207.02472). Submitted on 6 Jul 2022 ( v1 ), last revised 8 Mar 2024 (this version, v4)]
6. **M6** — Legg, Henry F. *Comment on "InAs-Al hybrid devices passing the topological gap protocol", Microsoft Quantum, Phys. Rev. B 107, 245423 (2023)*. Record date 2025/02/26; preprint (inspected record). [Primary record](https://arxiv.org/abs/2502.19560). Submitted on 26 Feb 2025]
7. **M7** — Anselmetti, G. L. R. et al. *End-to-end correlated subgap states in hybrid nanowires*. Record date 2019/08/15; preprint (inspected record). DOI: 10.1103/PhysRevB.100.205412. [Primary record](https://arxiv.org/abs/1908.05549). Submitted on 15 Aug 2019 ( v1 ), last revised 17 Sep 2019 (this version, v2)]
8. **M8** — Lai, Yi-Hua et al. *Presence versus absence of end-to-end nonlocal conductance correlations in Majorana nanowires: Majorana bound states versus Andreev bound states*. Record date 2019/01/09; preprint (inspected record). DOI: 10.1103/PhysRevB.100.045302. [Primary record](https://arxiv.org/abs/1901.02655). Submitted on 9 Jan 2019 ( v1 ), last revised 5 Jul 2019 (this version, v2)]
9. **M9** — Sarma, Sankar Das et al. *Disorder-induced zero-bias peaks in Majorana nanowires*. Record date 2021/03/09; preprint (inspected record). DOI: 10.1103/PhysRevB.103.195158. [Primary record](https://arxiv.org/abs/2103.05628v3). Submitted on 9 Mar 2021 ( v1 ), last revised 28 May 2021 (this version, v3)]
10. **M10** — Wang, Zhichuan et al. *Large Andreev bound state zero bias peaks in a weakly dissipative environment*. Record date 2022/02/18; preprint (inspected record). DOI: 10.1103/PhysRevB.106.205421. [Primary record](https://arxiv.org/abs/2202.09299v1). Submitted on 18 Feb 2022]

[Offline inspection and opt-in live reproduction](reproduce.md) · [Campaign overview](../summary.md) · [Separate tool QC](../findings.json)
