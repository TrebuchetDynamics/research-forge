# ResearchForge Skills

## Product skill

[`research-forge`](./skills/research-forge/SKILL.md) runs research through `rforge`:
literature and OSS search, screening, evidence, meta-analysis and review packages.
Claude Code, Codex, Pi and Hermes publish it with the five workflow skills in
[`adapters/skills/`](./adapters/skills/).

## Development skills

Development guidance comes from the installed [kit](https://github.com/TrebuchetDynamics/kit)
skills, such as `writing-for-agents` and `git-commit-push`. This repository keeps no
development skill set of its own.

TDD stays repository policy, not a skill: [CONTRIBUTING](./CONTRIBUTING.md) requires
red-green-refactor for every change, and `make check` is the gate. [AGENTS.md](./AGENTS.md)
carries the same rule for agents.

## Agent research depth contract

The [`research-forge`](./skills/research-forge/SKILL.md#search-depth-and-stopping) skill
starts at **Standard** depth (`--depth standard`); Quick is only for an explicit user
request. Escalate to Comprehensive when a Standard run ends with
`stop_reason: budget_exhausted` without coverage saturation, or when the user asks. A
completed sweep is not proof of exhaustive coverage, and `coverage_saturated` is a
reason to stop rather than fill a query quota.

Runs report depth, effective query/source/request caps, requests used, coverage,
remaining gaps and the engine's stop reason: `completed`, `budget_exhausted`,
`coverage_saturated`, `interrupted` or `error_threshold`. [CLI search budgets](./docs/cli.md#search-budgets-and-stop-reasons)
defines cap values and checkpoint accounting. Keep this guidance aligned when the
engine's stop semantics change.

## Development handoff format

When handing off a slice, include:

```text
TDD slice: <behavior>
Red: <test added and failing evidence>
Green: <implementation summary>
Refactor: <cleanup summary or none>
Validation: <commands/results>
Next slice: <recommended next failing test>
```
