---
name: research-forge-quality-security-tdd
description: Harden ResearchForge quality and security with tests first. Use for threat modeling, API key handling, dependency scanning, input validation, path safety, sandboxing external tools, CI gates, fuzzing, or vulnerability fixes.
---

# ResearchForge Quality and Security TDD

Use this skill whenever a slice touches untrusted input, files, network APIs, external commands, secrets, or CI hardening.

## Quick start

1. Identify the failure mode: leak, traversal, injection, unsafe command, flaky test, race, or bad dependency.
2. Write a failing regression/security test first.
3. Implement the smallest safe fix.
4. Add CI or lint coverage when the failure class can recur.
5. Record validation receipts.

## TDD contract

- **Red:** failing unit, integration, fuzz, race, or regression test.
- **Green:** minimal safe implementation.
- **Refactor:** centralize validation/sanitization and make unsafe states unrepresentable where practical.
- **Receipt:** targeted tests plus relevant security command.

## Focus areas

- API key and config secret handling.
- Path traversal in project/archive/clone/document paths.
- External command execution for git, GROBID, R, and converters.
- HTTP timeouts, retries, and rate-limit behavior.
- Input validation for imports and schemas.
- Fuzzing parsers/importers.
- Dependency and license scanning.
- Race tests for background jobs.
- CI gates: `go test`, `go vet`, staticcheck, govulncheck.

## Verification gate

Done requires:

- regression test fails before fix or is clearly shown as new coverage for a risk;
- secrets are not printed in logs/errors;
- unsafe paths/commands are rejected;
- CI command evidence is captured.

## Red lines

- Do not log tokens, keys, cookies, or full local private paths.
- Do not shell out with unsanitized user strings.
- Do not mark a vulnerability fixed without a regression test.
- Do not weaken CI to pass a feature.

## References

- [Security checklist](references/security-checklist.md)
