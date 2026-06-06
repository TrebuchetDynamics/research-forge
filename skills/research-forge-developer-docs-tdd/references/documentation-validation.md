# Documentation validation

Useful validation patterns:

- run CLI examples against a temp fixture project;
- generate command help and compare to checked-in docs;
- link-check Markdown files;
- verify docs do not mention commands that are absent from `rforge --help`;
- mark future/planned behavior clearly.

Handoff receipt:

```text
Docs changed: <files>
Behavior source: <tests/code/ADR>
Validation: <commands/results>
Known planned-only sections: <list or none>
```
