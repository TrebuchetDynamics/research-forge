---
name: research-forge-performance-tdd
description: Improve ResearchForge performance with benchmark-first TDD. Use for large libraries, deduplication speed, indexing throughput, parsing queues, UI responsiveness, memory use, caching, or API backoff performance.
---

# ResearchForge Performance TDD

Use this skill for performance work after behavior is correct or when a performance requirement blocks usability.

## Quick start

1. Define the performance symptom and target.
2. Add a failing benchmark, budget test, or measurable regression test.
3. Optimize the narrow bottleneck.
4. Confirm correctness tests still pass.
5. Record before/after evidence.

## TDD contract

- **Red:** benchmark or test that demonstrates the slow path, timeout, excessive allocation, or UI blocking risk.
- **Green:** smallest optimization that meets the target.
- **Refactor:** simplify data structures/concurrency without changing behavior.
- **Receipt:** before/after benchmark plus correctness tests.

## Focus areas

- Deduplication over thousands of records.
- Import/export of large BibTeX/CSV/JSON files.
- Query caching and API backoff.
- PDF parse job queues.
- Index rebuild throughput.
- Retrieval latency.
- Fyne UI responsiveness under background jobs.
- Report generation for large projects.
- Project archive/restore speed.

## Verification gate

Done requires:

- measurable before/after evidence;
- correctness tests remain green;
- no unbounded goroutine/channel/file growth;
- UI work remains off the main thread where applicable.

## Red lines

- Do not optimize by dropping provenance, auditability, or correctness.
- Do not add concurrency without cancellation/error propagation tests.
- Do not use live APIs for performance tests.

## References

- [Benchmark patterns](references/benchmark-patterns.md)
