---
name: lit
description: Use only when explicitly asked to run the ResearchForge lit workflow.
disable-model-invocation: true
---

# Literature synthesis

Load the `research-forge` methodology skill first, including its engine compatibility
check. It owns research decisions, provenance, verification and human gates.

For the requested corpus and output directory, make this engine call:

```sh
rforge lit --research-dir <research-dir> --out <output-dir>
```

Resolve missing inputs with the user and preserve their scope and restrictions.
Use the engine's results; this adapter supplies no research logic. If rforge is
missing or incompatible, follow `research-forge` installation guidance and stop.
