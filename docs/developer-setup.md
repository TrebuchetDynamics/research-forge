# Developer setup

```sh
make check
```

`make check` runs the default local gate: `go test ./...`, `go vet ./...`, the TODO owner-decision audit, and `git diff --check`.

License decision closeout has a live issue gate:

```sh
make license-decision-live-audit
make license-decision-approval-gate
```

`make license-decision-live-audit` prints the current issue #1 approval booleans. While owner approval is missing it reports `approved:false`; `make license-decision-approval-gate` must only pass with `approved:true` before adding `LICENSE` or checking off `TODO.md:34`.

For the CI security gate, install `govulncheck` and run:

```sh
govulncheck ./...
```

Development is TDD-only. Add a failing behavior test first, implement the smallest shared service/CLI behavior, then refactor.
