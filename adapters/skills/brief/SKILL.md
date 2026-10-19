---
name: brief
description: Use only when explicitly asked to run the ResearchForge brief workflow.
disable-model-invocation: true
---

# Research brief

Load the `research-forge` methodology skill first, including its engine compatibility
check. It owns research decisions, provenance, verification and human gates.

For the requested completed search directory and output file, make this engine call:

```sh
rforge brief <search-dir> --out <brief.md>
```

Resolve missing inputs with the user and preserve their scope and restrictions.
Use the engine's results; this adapter supplies no research logic. If rforge is
missing or incompatible, follow `research-forge` installation guidance and stop.
