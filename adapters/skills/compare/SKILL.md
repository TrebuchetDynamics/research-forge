---
name: compare
description: Use only when explicitly asked to run the ResearchForge compare workflow.
disable-model-invocation: true
---

# Research comparison

Load the `research-forge` methodology skill first, including its engine compatibility
check. It owns research decisions, provenance, verification and human gates.

For the requested completed search directories and output file, make this engine call:

```sh
rforge compare <search-dir-a> <search-dir-b> --out <comparison.md>
```

Resolve missing inputs with the user and preserve their scope and restrictions.
Use the engine's results; this adapter supplies no research logic. If rforge is
missing or incompatible, follow `research-forge` installation guidance and stop.
