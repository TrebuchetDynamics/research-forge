# Architecture checklist

Before changing architecture, ask:

- What behavior does this protect?
- Which package owns the domain rule?
- Which dependencies are external and need adapters?
- Can CLI and Fyne both use this service?
- How is provenance recorded?
- How will this be tested without network/services?
- Is the decision hard to reverse, surprising, and trade-off heavy enough for an ADR?

Preferred proof:

- contract tests for interfaces;
- compile-time interface assertions;
- package tests that use fakes;
- import-cycle-free `go test ./...`.
