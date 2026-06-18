# Improving open-source project search for ResearchForge

## Method and limits

I ran a Standard-depth ResearchForge sweep on open-source project discovery across GitHub, GitLab, repository mining, code search, package registries, dependency graphs, and software supply-chain signals. Sources were OpenAlex, Crossref, Semantic Scholar, and arXiv. The sweep returned 280 raw records and 237 DOI/title-deduplicated records. Crossref had six normalization failures, Semantic Scholar rate-limited three searches, and OpenAlex had one normalization failure; these are recorded in `failures.jsonl`.

No repository clones, package downloads, or gated integration actions were performed. This report uses metadata, titles, venues, DOI records, and citation expansion only.

## Bottom line

ResearchForge should improve OSS project search by moving from single-forge GitHub lookup to a multi-provider discovery plan: code forges, package registries, archival indexes, dependency/security databases, and research-software archives. Ranking should combine maintenance, license, activity, dependency risk, archive presence, documentation, package metadata, and relevance; it should not use GitHub stars alone.

## Main themes

### 1. Search across forges, not only GitHub

The literature and project landscape support broadening beyond GitHub. **"World of Code: Enabling a Research Workflow for Mining and Analyzing the Universe of Open Source VCS data"** (`10.48550/arxiv.2010.16196`) supports a cross-repository/cross-origin view of open-source version-control data. ResearchForge should therefore consider GitHub, GitLab, Codeberg/Forgejo, SourceHut, Bitbucket, self-hosted Gitea/Forgejo, and Software Heritage origins.

Implementation implication: add a provider plan or discovery layer that produces provider-specific URLs/API calls and records provenance per provider before any clone or dependency decision.

### 2. Stars are useful but unsafe as the only ranking feature

**"What's in a GitHub Star? Understanding Repository Starring Practices in a Social Coding Platform"** (`10.1016/j.jss.2018.09.016`) shows stars carry social/relevance signals, but they should be interpreted alongside activity, issue health, documentation, and fit. **"Predicting the Popularity of GitHub Repositories"** (`10.1145/2972958.2972966`) supports feature-based popularity modeling, but popularity is not the same as suitability or safety.

Implementation implication: ResearchForge should expose stars/forks/watchers as one signal, never as the final ranking. Use a scorecard that separates popularity, maintenance, security, licensing, and domain fit.

### 3. Maintenance and project health are first-class search filters

**"Is this GitHub Project Maintained? Measuring the Level of Maintenance Activity of Open-Source Projects"** (`10.1016/j.infsof.2020.106274`) directly supports maintenance signals. Candidate discovery should include recent commits, release cadence, issue response, PR merge activity, stale issue backlog, CI status, and archived/read-only status.

Implementation implication: add provider metadata fields for `lastActivity`, `releaseCadence`, `openIssues`, `defaultBranchUpdated`, `archived`, `ciDetected`, and `securityPolicy`.

### 4. Repository curation matters at GitHub scale

**"PHANTOM: Curating GitHub for engineered software projects using time-series clustering"** (`10.1007/s10664-020-09825-8`) supports filtering and curation before analysis. Large forge search results contain demos, abandoned repos, forks, generated repositories, homework, and mirrors.

Implementation implication: ResearchForge should support exclusion filters for forks, mirrors, toy repos, inactive projects, unlicensed repos, generated code, and projects below a minimum activity threshold. For research, every exclusion should be provenance-recorded.

### 5. Package registries add dependency and ecosystem signals

Forge search misses packages whose repository metadata is incomplete or hosted elsewhere. Project search should query package registries: pkg.go.dev, crates.io, PyPI, npm, Maven Central, RubyGems, conda-forge, Docker Hub, and domain-specific registries where relevant.

Implementation implication: use package registries to enrich candidates with releases, dependents, downloads, declared license, repository link, vulnerability/advisory links, and ecosystem-specific quality signals.

### 6. Supply-chain security should influence search and selection

**"Backstabber’s Knife Collection: A Review of Open Source Software Supply Chain Attacks"** (`10.1007/978-3-030-52683-2_2`) and **"Pinning Is Futile: You Need More Than Local Dependency Versioning to Defend against Supply Chain Attacks"** (`10.1145/3715728`) support adding security and dependency-risk checks. A good project search tool should surface OpenSSF Scorecard, package signing, SBOM availability, dependency freshness, known vulnerabilities, release provenance, and maintainer risk indicators.

Implementation implication: separate discovery from approval. Agents may collect signals, but humans should approve integration/dependency use.

### 7. Semantic code/repository search can improve relevance

**"Big Code Search: A Bibliography"** (`10.1145/3604905`) and **"GraphSearchNet: Enhancing GNNs via Capturing Global Dependencies for Semantic Code Search"** (`10.1109/tse.2022.3233901`) support semantic/code-aware retrieval. ResearchForge can start with metadata search and later add embeddings/code-symbol search across cloned or indexed candidates.

Implementation implication: staged approach: metadata search first, then optional local clone/index for shortlisted candidates, with explicit clone approval and licensing review.

## Recommended ResearchForge feature design

1. `rforge oss search-plan --query <text> [--ecosystem ...]`: deterministic provider plan across forges, registries, archives, and security databases.
2. `rforge oss search --provider github|gitlab|codeberg|sourcehut|pkg-go|pypi|npm|crates --query <text> --out candidates.json`: live provider adapters, each with provenance and rate-limit metadata.
3. Candidate model fields: source provider, repo URL, package URL, license, stars/forks/downloads, last activity, release cadence, issues/PR activity, CI detected, security policy, dependency metadata, archive status, topics/tags, description, language, and provenance refs.
4. Ranking model: separate relevance, maintenance, security, popularity, ecosystem fit, and license/shareability sub-scores.
5. Human gates: dependency/import approval, clone approval for large repos, license review, and integration disposition (`pattern-reference`, `adapter-only`, `integrate`, `avoid`).
6. Dashboard: show provider coverage, candidate clusters, stale/risky flags, and why each project was ranked.

## Performance claims hygiene

Do not claim that stars, downloads, or recent commits prove quality. Cite exact evidence:

- Use `10.1016/j.jss.2018.09.016` when discussing GitHub stars as social signals.
- Use `10.1016/j.infsof.2020.106274` when discussing maintenance/activity measurement.
- Use `10.1007/s10664-020-09825-8` when discussing GitHub-scale curation/filtering.
- Use `10.48550/arxiv.2010.16196` when discussing cross-origin repository corpora.
- Use `10.1007/978-3-030-52683-2_2` and `10.1145/3715728` when discussing supply-chain/security risk.

## Evidence gaps

- GitLab/Codeberg/SourceHut-specific academic coverage was thinner than GitHub coverage.
- Package registry quality signals vary substantially by ecosystem.
- Some relevant records lacked DOI metadata in the sweep.
- Live API behavior for GitLab/Codeberg/SourceHut/package registries was not tested in this research run.

## Concrete implementation slice completed from this research

Added a deterministic `rforge oss search-plan` command that prints a provider plan across GitHub, GitLab, Codeberg, SourceHut, Software Heritage, OpenSSF Scorecard, ecosystem package registries, and research archives. This is intentionally a planning step only: it does not clone repositories, import dependencies, or self-approve integration decisions.
