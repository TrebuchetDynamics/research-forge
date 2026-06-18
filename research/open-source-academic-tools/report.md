# Open-source research tools and academic workflows: implications for making ResearchForge wider and better

## Method and limits

Question interpreted as: how can ResearchForge learn from open-source research tools and academic evidence-synthesis workflows to broaden coverage, improve trust, and serve academics better?

I ran a standard-depth rforge sweep across OpenAlex, Crossref, Semantic Scholar, and arXiv using six query families: systematic-review tools, open-science infrastructure, reproducible evidence synthesis, reference-management workflows, active-learning screening, and full-text parsing. No copyrighted full text was downloaded; conclusions below use retrieved metadata/search results and citation graphs only.

Source coverage from `rforge search stats --dir .`:

- arXiv: 120 records / 6 files
- Crossref: 120 records / 6 files
- OpenAlex: 73 records / 6 files
- Semantic Scholar: 16 records / 6 files; several queries returned HTTP 429
- Total unique DOIs: 310

Citation expansion was run for five anchor papers: ASReview, systematic-review automation, machine-learning guide for evidence synthesis, S2ORC, and PaperMage.

## Bottom line

The strongest direction for ResearchForge is not to become one more isolated literature-search CLI; it should become a reproducible evidence-synthesis workbench that connects discovery, dedupe, legal full-text acquisition, parser arbitration, screening, extraction, statistics, and packaging under explicit human review gates. The retrieved literature supports widening ResearchForge through governed adapters to proven open infrastructure, while improving it through auditability, reviewer decision trails, and reproducible artifacts.

## Main themes

### 1. Treat systematic-review automation as the spine

Several retrieved papers frame systematic review automation as a workflow problem, not a single model problem:

- `10.1186/2046-4053-3-74`, **Systematic review automation technologies**, Systematic Reviews.
- `10.1186/s13643-019-1074-9`, **Toward systematic review automation: a practical guide to using machine learning tools in research synthesis**, Systematic Reviews.
- `10.1186/s12874-022-01805-4`, **Machine learning computational tools to assist the performance of systematic reviews: A mapping review**, BMC Medical Research Methodology.

Implication for rforge: prioritize an end-to-end `rforge forge` or review-spine workflow over isolated subcommands. Every step should leave durable state: query plan, source records, dedupe decisions, screening queues, extraction schemas, analysis run manifests, and report/package checksums.

### 2. Make active-learning screening auditable, not magical

The ASReview anchor paper, `10.1038/s42256-020-00287-7`, **An open source machine learning framework for efficient and transparent systematic reviews**, Nature Machine Intelligence, is directly aligned with ResearchForge's desired stance: ML can reduce screening workload, but must remain transparent and reviewer-controlled.

Implication for rforge: add ASReview/revtools/RobotReviewer-style assistance as suggestion queues, not automatic inclusion/exclusion. Useful slices:

- uncertainty-ranked title/abstract screening queues;
- exploration vs. exploitation settings recorded in the lockfile;
- reviewer decision exports;
- stopping diagnostics and recall-risk warnings;
- reproducible model/version configuration.

### 3. Widen scholarly graph/source coverage through adapters

The open-science infrastructure sweep surfaced graph and corpus papers relevant to source breadth:

- `10.18653/v1/2020.acl-main.447`, **S2ORC: The Semantic Scholar Open Research Corpus**, ACL 2020.
- `10.1007/s11192-019-03217-6`, **Software review: COCI, the OpenCitations Index of Crossref open DOI-to-DOI citations**, Scientometrics.
- `10.1007/s11192-020-03690-4`, **Google Scholar, Microsoft Academic, Scopus, Dimensions, Web of Science, and OpenCitations' COCI: a multidisciplinary comparison of coverage via citations**, Scientometrics.
- `10.1007/s40747-022-00806-6`, **Scholarly knowledge graphs through structuring scholarly communication: a review**, Scientometrics.

Implication for rforge: use a capability registry for sources. Each adapter should declare IDs, cursor/resume support, rate limits, citation/reference support, license/OA metadata, field coverage, and known failure modes. This will make rforge wider without hiding source-specific limits.

### 4. Build parser arbitration, not parser monoculture

Full-text parsing search found `10.18653/v1/2023.emnlp-demo.45`, **PaperMage: A Unified Toolkit for Processing, Representing, and Manipulating Visually-Rich Scientific Documents**, EMNLP Demo 2023, plus GROBID-related records. The existing `opensource/inventory/` already tracks GROBID, CERMINE, Science Parse, S2ORC doc2json, Anystyle, and PaperMage.

Implication for rforge: parser output should be compared and adjudicated. Store raw parser outputs, stable passage IDs, offsets where possible, extracted references, confidence/risk, and conflicts requiring reviewer review. A good rforge differentiator would be a `parse compare` or `parser arbitration` report.

### 5. Academic users need reference-manager interoperability

The reference-management sweep found Zotero/JabRef records, including:

- `10.1179/2047480614z.000000000190`, **Zotero: A free and open-source reference manager**.
- `10.1080/1941126x.2024.2417123`, **Zotero: A Highly Functional, Open-Source Reference Management Software**.
- `10.5121/csit.2020.101121`, **A Case Study on Maintainability of Open Source Software System Jabref**.

Implication for rforge: academics will trust rforge faster if it imports/exports their existing libraries cleanly. High-value work: Zotero/JabRef collection mapping, BibTeX/BibLaTeX cleanup diffs, citation-key preservation, annotation import/export, duplicate groups, and linked-file privacy checks.

### 6. Software citation and provenance should be first-class academic output

The sweep surfaced `10.7717/peerj-cs.86`, **Software citation principles**, PeerJ Computer Science. This supports ResearchForge's provenance-first posture.

Implication for rforge: reports and reproducible packages should cite software/tools as well as papers. Include tool versions, source API versions where available, parser/model versions, settings, and checksums in report appendices.

## Performance claims hygiene

Do not make blanket claims such as "active learning reduces screening by X%" or "parser Y is best" without tying the number to a named paper, dataset, and workflow. ResearchForge should present such claims as source-specific, review-specific diagnostics: e.g. the ASReview paper `10.1038/s42256-020-00287-7` supports transparent ML-aided screening as a direction, but rforge should still require project-local validation, recall-risk reporting, and reviewer confirmation.

## Evidence gaps

- Semantic Scholar returned HTTP 429 for several queries, limiting coverage from that source in this run.
- Searches retrieved metadata only; no full-text methods sections or benchmark tables were inspected.
- Reference-manager workflow evidence was thinner than screening and source-graph evidence; more targeted searches around Zotero/JabRef APIs and academic library workflows would help.
- The run did not benchmark actual OSS tools locally; it only identifies source-backed directions for ResearchForge.

## Concrete ResearchForge roadmap recommendations

1. **Ship a reproducible review package path**: manifest, lockfile, query plan, source results, dedupe log, screening audit, extraction schema, evidence table, analysis artifacts, report, redaction policy, checksums.
2. **Add adapter capability registry**: OpenAlex, Crossref, Semantic Scholar, arXiv, PubMed, Europe PMC, OpenCitations/COCI, S2ORC, NASA ADS, DOAJ/CORE.
3. **Deepen reference-manager integration**: Zotero and JabRef import/export with collection, tag, note, annotation, linked-file privacy, and citation-key fidelity.
4. **Implement parser arbitration**: GROBID + S2ORC-style JSON + PaperMage + CERMINE/Anystyle fallback, with reviewer-visible conflicts.
5. **Add auditable screening assistance**: ASReview/revtools-style queues, uncertainty sampling, dedupe/cluster review, stopping diagnostics, and reviewer decision exports.
6. **Govern NLP/retrieval adapters**: OpenSearch/Qdrant/SentenceTransformers/KeyBERT/SciSpaCy/BERTopic should have privacy profiles, model locks, evaluation fixtures, and citation-linked suggestion queues.
7. **Make academic trust visible in the UI**: show provenance, source coverage, rate-limit failures, missing identifiers, human gates, and reproducibility status as first-class cockpit panels.

## Suggested next inventory additions

The existing `opensource/inventory/` already covers many important tools. Based on this run, likely additions or deeper notes include OpenCitations/COCI, S2ORC as corpus/source infrastructure distinct from doc2json, Unpaywall/OA discovery if not already separate, Covidence/Rayyan as proprietary workflow comparators, and Cochrane tooling/guidance as workflow-pattern references.
