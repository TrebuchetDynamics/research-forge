# Developer setup

```sh
make check
```

`make check` runs the default local gate: `go test ./...`, `go vet ./...`, the TODO owner-decision audit, and `git diff --check`.

For the CI security gate, install `govulncheck` and run:

```sh
govulncheck ./...
```

Development is TDD-only. Add a failing behavior test first, implement the smallest shared service/CLI behavior, then refactor.
