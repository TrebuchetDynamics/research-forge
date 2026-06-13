# Performance notes

Benchmark coverage includes datasets/scenarios for 10, 1,000, and 100,000 records where practical:

- deduplication benchmarks in `internal/library`;
- import/export benchmarks in `internal/library`;
- index rebuild benchmark in `internal/retrieval`;
- report generation benchmark in `internal/report`.

Long-running jobs must accept cancellation. Future web GUI jobs should run through background job abstractions to keep UI responsive.
