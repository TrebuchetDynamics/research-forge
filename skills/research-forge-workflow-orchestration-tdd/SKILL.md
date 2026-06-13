---
name: research-forge-workflow-orchestration-tdd
description: Plan and execute ResearchForge vertical slices with TDD across skills. Use for milestone breakdown, issue creation, slice sequencing, handoffs, acceptance criteria, or coordinating CLI/UI/domain/test work.
---

# ResearchForge Workflow Orchestration TDD

Use this skill to break ResearchForge work into test-first vertical slices and route to specialist skills.

## Quick start

1. Read `DEVELOPMENT_PLAN.md`, `SKILLS.md`, and current git status.
2. Choose the smallest user-visible behavior that advances a milestone.
3. Define the failing test first and the specialist skill to use.
4. Execute or hand off the slice with a red-green-refactor contract.
5. Stop only at a validated checkpoint, blocker, or owner decision.

## Slice template

```text
Slice: <one behavior>
Milestone: <0-8>
Primary skill: <skill name>
Red test: <test to write first>
Green implementation: <minimal code>
Refactor target: <cleanup after green>
Validation: <commands>
Acceptance: <observable result>
Next slice: <dependent behavior>
```

## Routing map

- Foundation/package seams -> `research-forge-foundation-tdd` or `research-forge-architecture-tdd`.
- Source connectors/library -> `research-forge-scholarly-ingestion-tdd`.
- Fixtures/harnesses -> `research-forge-test-fixtures-tdd`.
- OSS study -> `research-forge-oss-intelligence-tdd`.
- Documents/parsing/indexing -> `research-forge-document-pipeline-tdd`.
- Screening -> `research-forge-screening-tdd`.
- Evidence -> `research-forge-evidence-extraction-tdd`.
- Analysis -> `research-forge-meta-analysis-tdd`.
- Reports -> `research-forge-reporting-tdd`.
- web GUI -> `research-forge-web-ui-tdd`.
- Data/privacy/compatibility -> `research-forge-data-governance-tdd`.
- Security/quality -> `research-forge-quality-security-tdd`.
- Performance -> `research-forge-performance-tdd`.
- Release/docs -> `research-forge-release-packaging-tdd` or `research-forge-developer-docs-tdd`.

## Verification gate

Done requires:

- each planned slice has a red test named;
- dependencies are ordered;
- risky owner decisions are called out;
- completed slices include validation receipts.

## Red lines

- Do not create broad implementation batches without tests per slice.
- Do not route risky architecture decisions into code without owner decision or ADR review.
- Do not mark a milestone complete without mapping acceptance criteria to evidence.

## References

- [Slice planning guide](references/slice-planning-guide.md)
