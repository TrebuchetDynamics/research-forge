# Developer setup

## Core prerequisites

Before running the local gate, make these available on `PATH`:

- **Go 1.26** minimum; [`go.mod`](../go.mod) separately pins toolchain version **1.26.6**.
- **Git** for repository operations and Git-backed tests.
- **GNU Make** for the targets in [`Makefile`](../Makefile), with a POSIX shell and standard command-line utilities.
- **Node.js >=22**, as required by the root [`package.json`](../package.json), for the adapter tests.

On a cold cache, provision the Go toolchain and module dependencies before an offline run. Go may otherwise download the selected toolchain and missing modules (`go mod download` can prefetch the modules while network access is available). Normal tests use offline fixtures; provisioning is separate from test execution. The root Node adapter tests need Node itself, not a site dependency install.

## Core local gate

```sh
make check
```

`make check` runs the default local gate defined in [`Makefile`](../Makefile): `gofmt` check, `go mod tidy -diff`, `go test ./...`, `node --test adapters/test`, `go vet ./...`, `todo-completion-audit`, inventory check, and `git diff --check`. The TODO completion audit prints active decisions and runs `rforge decisions --completion-audit CHECKLIST.md docs/todo-completion-audit.md`; it is not just the unchecked-item audit from `make todo-audit`.

## Optional documentation site

The Astro site under `site/` is separate from the core gate and the Go + HTMX local web GUI. Its locked Astro version requires **Node.js >=22.12** and **npm >=9.6.5** (see [`site/package-lock.json`](../site/package-lock.json)); the core Node >=22 floor alone is not sufficient for site work.

For site work, install its locked dependencies with `npm ci` in `site/`, then use `npm run dev` or `npm run build` there (scripts are in [`site/package.json`](../site/package.json)). A cold npm cache needs network access to provision those dependencies before offline builds. Site installation/build is not part of `make check` and is not required for core contributor setup.

## License approval revalidation

The `project_license` decision was resolved as MIT on 2026-06-13, and `LICENSE` is present. The [owner-decision record](owner-decisions.md#project-license) preserves the approval evidence. Licensing is not an outstanding onboarding blocker, and no live approval check is needed for routine offline setup.

The live issue gates remain required for license changes and release revalidation, not as a claim of fresh approval from this guide:

```sh
make license-decision-live-audit
make license-decision-approval-gate
```

These commands require the GitHub CLI (`gh`) and authorized live access to issue #1. `make license-decision-live-audit` prints the issue's current approval booleans; `approved:false` indicates missing approval fields, not the recorded decision's current status. `make license-decision-approval-gate` must pass with `approved:true` for license/release approval. The original prerequisite before adding `LICENSE` or checking off `CHECKLIST.md:34` is retained in the historical record; future releases must still revalidate it. See the [1.0 release gate](1.0-release.md) for the required fields and remaining criteria. No live audit was run for this documentation update.

## Security and development workflow

Run the CI security gate with:

```sh
make vuln
```

The Makefile pins the scanner binary for reproducible runs; `govulncheck` still queries the live Go vulnerability database by default.

Development is TDD-only. Add a failing behavior test first, implement the smallest shared service/CLI behavior, then refactor.
