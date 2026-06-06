---
name: research-forge-data-governance-tdd
description: Build ResearchForge project data governance with strict TDD. Use for schemas, migrations, project archives, privacy defaults, copyright/OA policy, provenance retention, lockfile compatibility, or data import/export safety.
---

# ResearchForge Data Governance TDD

Use this skill for data safety, compatibility, privacy, copyright, and reproducibility guarantees.

## Quick start

1. Read PRD sections 8.1-8.9 and `DEVELOPMENT_PLAN.md` data-related milestones.
2. Choose one data guarantee.
3. Write a failing test that would catch data loss, privacy leakage, or reproducibility drift.
4. Implement minimal validation, policy, migration, or export behavior.
5. Refactor while keeping backward compatibility explicit.

## TDD contract

- **Red:** failing test for schema migration, archive restore, privacy redaction, copyright guard, lockfile update, checksum, or provenance retention.
- **Green:** smallest data-policy implementation.
- **Refactor:** centralize policy checks and keep formats deterministic.
- **Receipt:** run targeted tests and any archive/migration smoke command.

## Slice areas

- Project manifest validation.
- Lockfile versioning and tool digests.
- Database migrations and rollback safety.
- Project archive/restore.
- Redaction for shareable exports.
- Legal OA asset policy.
- Checksums for assets, scripts, reports, and analysis outputs.
- Provenance retention and compaction rules.
- Import/export format compatibility.

## Verification gate

Done requires:

- test proves the governance rule;
- fixtures include old/current format examples where compatibility matters;
- user-sensitive fields are redacted in shareable paths;
- copyright-sensitive assets are not committed or exported accidentally.

## Red lines

- Do not silently migrate or delete user data without backup/receipt.
- Do not export private notes, API keys, or local paths in shareable reports.
- Do not treat full-text availability as legal permission unless OA/license metadata supports it.

## References

- [Governance scenarios](references/governance-scenarios.md)
