---
name: research-forge-screening-tdd
description: Build ResearchForge systematic-review screening with strict TDD. Use for include/exclude decisions, reason tags, reviewer workflows, conflict queues, ASReview-style prioritization, PRISMA counts, or screening UI.
---

# ResearchForge Screening TDD

Use this skill for Milestone 4 screening workflows.

## Quick start

1. Read `DEVELOPMENT_PLAN.md` Milestone 4 and PRD section 7.5.
2. Select one screening behavior.
3. Write a failing test for decision state, counts, or queue behavior.
4. Implement the minimal domain and CLI behavior.
5. Refactor without changing audit semantics.

## TDD contract

- **Red:** failing test for decision recording, reason validation, reviewer attribution, queue filtering, conflict handling, or PRISMA count generation.
- **Green:** minimal production code to pass.
- **Refactor:** keep screening state derivable from stored decisions/events.
- **Receipt:** run targeted tests and CLI smoke commands.

## Slice order

1. Screening stages and decision enums.
2. Exclusion reason configuration.
3. `rforge screen decide`.
4. Reviewer attribution and timestamps.
5. Queue filtering by stage/status.
6. Conflict/uncertain queue.
7. PRISMA counts from event history.
8. CSV import/export.
9. Active-learning prioritization scaffold.
10. web GUI screening queue view model.

## Verification gate

Done requires:

- tests prove rejected reason tags are not accepted;
- decisions are auditable and attributable;
- PRISMA counts are regenerated, not hand-authored;
- CLI output is deterministic.

## Red lines

- Do not overwrite prior screening decisions without preserving history.
- Do not use active learning to hide records from review.
- Do not make reviewer identity ambiguous in multi-reviewer workflows.

## References

- [Screening scenarios](references/screening-scenarios.md)
