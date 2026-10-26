# Documentation map

Start here to choose a guide and understand its authority. This is a category map, not a second command reference or a claim that every dated document is current.

- **Current guides** explain supported use and development; check the installed engine's help/capabilities for executable behavior.
- **Specifications and decisions** own requirements, formats and accepted choices. A plan or acceptance criterion is not proof of implementation or qualification.
- **Reference and historical evidence** preserve observations at their recorded revision, date, inputs and environment. They do not establish current external facts, scientific truth or fresh live verification.

## Start and use

| Need | Read first | Authority / limits |
| --- | --- | --- |
| Choose a workflow | [Repository overview](../README.md), [quickstart](quickstart.md) | Current entry points; source queries require network access. |
| Install the engine or agent integration | [Installation](install.md), [agent installation](agent-install.md) | Current installation, update, removal and host-specific setup owners. Agent setup is distinct from CLI-only use. |
| Commands, search folders and MCP | [CLI reference](cli.md), especially [search budgets](cli.md#search-budgets-and-stop-reasons) and [MCP stdio server](cli.md#mcp-stdio-server) | Command/artifact reference; implementation and installed command contracts determine executable behavior. |
| Project files and guided reviews | [Project format](project-format.md), [forge state machine](rforge-forge-state-machine.md) | Project artifact and workflow contracts; a search directory is not a project store. |
| Review-package contents and checks | [Reproducible review package](reproducible-review-package.md) | Package format, privacy, audit and replay contract; not scientific approval. |
| Browser documentation | [Site overview](../site/src/content/docs/index.mdx), [getting started](../site/src/content/docs/getting-started.mdx), [workflow](../site/src/content/docs/workflow.mdx), [CLI cheatsheet](../site/src/content/docs/cli.mdx), [sources](../site/src/content/docs/sources.mdx), [packages](../site/src/content/docs/packages.mdx), [agent skill](../site/src/content/docs/agent-skill.mdx) | The seven `site/` MDX guides are the public presentation layer; use the detailed owners above rather than treating a cheatsheet as the full contract. |

## Build and contribute

| Area | Entry points | Authority / limits |
| --- | --- | --- |
| Contributor workflow | [Contributing](../CONTRIBUTING.md), [developer setup](developer-setup.md), [agent instructions](../AGENTS.md), [code of conduct](../CODE_OF_CONDUCT.md), [PR template](../.github/PULL_REQUEST_TEMPLATE.md) | Repository development instructions and contribution expectations, not research-user setup. |
| Source tour and vocabulary | [Architecture](architecture.md), [glossary](../CONTEXT.md) | Current package ownership and stable terminology; decisions belong in ADRs. |
| Developer skills | [Skill catalog](../SKILLS.md), [developer-docs skill](../skills/research-forge-developer-docs-tdd/SKILL.md) | `skills/research-forge-*-tdd/` and their `references/` are executable development guidance, not the public research pack. |
| Public agent instructions | [Research methodology skill](../skills/research-forge/SKILL.md), [workflow skills](../adapters/skills/), [Hermes skill mirror](../hermes-plugin/skills/research-forge/SKILL.md) | Instructions consumed by hosts. `adapters/skills/` contains the primary mirror plus five explicit workflows; `hermes-plugin/` is another host integration. Do not bulk-install the developer catalog for users. |
| Examples and test data | [Open-data example](../examples/open-data-project/README.md), [tiny benchmark fixture](../internal/benchmarks/testdata/fixture-tiny/review.md), [fixture policy](fixture-policy.md) | `examples/` and documentation under `internal/` describe examples/fixtures, not live research receipts. |
| Release work | [Release packaging](release.md), [release-notes template](release-notes-template.md), [Claude release skill](../.claude/skills/rforge-release/SKILL.md), [Pi release skill](../.pi/skills/rforge-release/SKILL.md) | Release procedures and executable instructions, not permission to publish or evidence of a completed release. |

## Trust and qualification

- [Privacy and copyright](privacy-copyright.md), [threat model](threat-model.md) and [outbound API data flow](outbound-api-data-flow.md) own trust/egress guidance.
- [Claim-evidence schema](claim-evidence-schema.md) defines separate verification facets and provenance levels; no single status establishes that a scientific conclusion is true.
- [External services](external-services.md) describes optional service setup; [source connectors](source-connectors.md) records backlog, terms-review and implementation gates, not blanket upstream approval.
- [Live smoke procedures](live-smoke.md), [host qualification](qualification/hosts.md) and its [receipt template](qualification/host-receipt-template.md) describe checks to execute. A template, offline test or historical receipt is not a fresh host/provider pass.
- [Performance notes](performance.md) and [benchmark specification](benchmark/spec.md) distinguish measurement scope from improvement claims. Read the [dated baseline](benchmark/baseline-2026-10-19.md) with its locked inputs and limitations.
- Release criteria remain in [pre-alpha](pre-alpha-release.md), [alpha](alpha-release.md), [beta](beta-release.md) and [1.0](1.0-release.md) gates. These documents are criteria/records, not a declaration of present readiness; delivery claims require release-specific evidence.

## Specifications, decisions and plans

| Owner | How to read it |
| --- | --- |
| [Product requirements](../RESEARCH-FORGE-PRD.md), [roadmap](../ROADMAP.md), [development plan](../DEVELOPMENT_PLAN.md) | Product intent and sequencing, not an availability checklist. |
| [ADRs](adr/README.md) and [ADR directory](adr/) | Durable choices; read each record's status, including screening order and verification/provenance decisions. This map does not amend them. |
| [Owner decisions](owner-decisions.md), [decision resolution checklist](decision-resolution-checklist.md), [license decision](license-decision.md), [decision issue records](decisions/) | Decision process and retained decision evidence; historical issue scaffolds do not imply an active blocker. |
| [Meta-analysis spine blueprint](meta-analysis-spine-blueprint.md), [roadmap](meta-analysis-spine-roadmap.md), [acceptance matrix](meta-analysis-spine-acceptance-matrix.md) | Design/acceptance references; verify implemented slices against source/tests. |
| [Web GUI plan](web-gui-plan.md), [superseded Fyne plan](fyne-desktop-plan.md), [verifiable-engine plan](verifiable-engine-plan.md), [Hermes installer plan](hermes-installer-plan.md), [design specs](superpowers/specs/) | Dated plans and design history. Preserve supersession/status labels; do not infer current runtime behavior from a planned seam. |
| [Checklist](../CHECKLIST.md), [remaining-item audit](remaining-todo-audit.md), [completion audit](todo-completion-audit.md) | Checklist and closeout evidence, distinct from the executable task store. |
| [Task store](../tasks/), [goals](../tasks/goals/), [task contracts](../tasks/items/), [receipts](../tasks/evidence/) | Coordinator-owned records. Read status, closure and receipt qualification; source notes may use repository-root paths rather than rendered relative links. Workers do not rewrite these as ordinary docs. |
| [Planning and evidence handoffs](tasks/), including the [documentation ROI plan and inventory](tasks/documentation-roi-plan.md#all-docs-inventory-and-coverage) | `docs/tasks/` holds source plans/evidence reviews, not the runnable queue. [Blockers](../BLOCKERS.md) holds owner-input coordination; [frozen TODO archive](../TODO.archive.md) is read-only history. |

## Reference and historical corpus

| Area | Entry points | Limits |
| --- | --- | --- |
| Research reports and captured results | [Research corpus](../research/), [citation notes](../research/CITATIONS.md), [scientific examples and limitations](../research/frontier-science/README.md) | Dated reports, results, raw artifacts, authorizations and receipts retain their original scope. They are not independently re-reviewed science or current service availability claims. |
| Domain-specific reference notes | [LightGBM leakage review](lightgbm-crypto-leakage-review.md), [training tips](lightgbm-price-data-training-tips.md) | Research/reference notes under `docs/`, not core product guarantees or general scientific validation. |
| OSS studies | [OSS study policy](../opensource/README.md), [inventory index](../opensource/inventory/README.md) | `opensource/` records pattern references, licenses/risks and integration dispositions. Listing a project is not dependency approval or fresh verification of upstream facts. |
| Evaluation contracts and retained runs | [Benchmark docs](benchmark/), [agent-tier contract](benchmark/agent-tier.md), [pilot](benchmark/pilot-pi-2026-10-21.md), [frontier plan](benchmark/frontier-science-plan.md), [runbook](benchmark/frontier-science/runbook.md) | Separate protocols/criteria from dated measurements and live qualification. |
| Benchmark tooling and inputs | [Agent-tier tooling](../benchmarks/agent-tier/README.md), [live lane](../benchmarks/agent-tier/live/README.md), [report deltas](../benchmarks/report-delta/README.md), [candidate sets](../benchmarks/sets/candidates/README.md), [locked pool review](../benchmarks/sets/pool-a/review.md) | `benchmarks/` includes tooling, unlocked candidates and locked evaluation inputs. Preserve bodies/checksums; stale historical links do not authorize rewriting locked evidence. |

**RF-BENCH remains paused and unmet** under the [2026-10-23 owner decision](tasks/rf-bench-offline-round-2.md#owner-decision-planning-remains-paused). Navigation or documentation maintenance neither reopens planning nor authorizes live work, credentials, new inputs or spend.

Together these categories cover the saved inventory's root, `docs/`, `site/`, `skills/`, `adapters/`, `hermes-plugin/`, `research/`, `benchmarks/`, `opensource/`, `examples/`, `internal/`, `tasks/`, `.claude/`, `.pi/` and `.github/` areas. The [287-file baseline inventory](tasks/documentation-roi-plan.md#all-docs-inventory-and-coverage) is a historical count, not a second catalog to maintain here.

## Maintenance and update triggers

- When commands, flags, search outputs or MCP tools change, update the [CLI reference](cli.md) and affected introductory/site guides; check against implementation and installed help/capabilities.
- When installation/host packaging changes, update [installation](install.md), [agent installation](agent-install.md) and affected public skill mirrors together; developer skills remain separate.
- When package/interface ownership changes, refresh the [source tour](architecture.md); when artifacts or trust boundaries change, update [project format](project-format.md), [package contract](reproducible-review-package.md) and affected trust references. Durable decision changes need their own ADR process, not a silent edit to history.
- When a documentation area is added, moved or retired, update this category map and its README entry. Preserve dated evidence and link its qualifications rather than regenerating historical reports or receipts.
- Validate changed local targets **and fragments**, reconcile the affected area with the inventory, and review the diff for accidental instruction/authority changes. Use the existing offline docs guard: `env -u GIT_DIR -u GIT_WORK_TREE -u GIT_INDEX_FILE go test ./internal/docs -count=1`, plus `git diff --check`. The repository-wide gate is `make check`; Autowork's coordinator runs it on the merged tree. These checks do not execute live tutorials or qualify science, hosts, providers or releases.
