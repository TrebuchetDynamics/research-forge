# Benchmark patterns

Prefer synthetic deterministic datasets:

- 10, 1,000, and 100,000 paper records for dedupe/index tests;
- repeated fixture documents for parser queue tests;
- generated evidence tables for report tests.

Useful commands:

```sh
go test ./... -run TestName
go test ./internal/library -bench Dedup -benchmem
go test ./internal/reports -bench Report -benchmem
```

For regressions, record:

- dataset size;
- command;
- baseline timing/allocation;
- new timing/allocation;
- correctness test receipt.
