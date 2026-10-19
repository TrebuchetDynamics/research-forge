---
name: paper
description: Use only when explicitly asked to run the ResearchForge paper workflow.
disable-model-invocation: true
---

# Paper resolution

Load the `research-forge` methodology skill first, including its engine compatibility
check. It owns research decisions, provenance, verification and human gates.

For the requested DOI, arXiv ID, OpenAlex ID or title, make this engine call:

```sh
rforge paper <identifier-or-title>
```

Resolve missing inputs with the user and preserve their scope and restrictions.
Use the engine's results; this adapter supplies no research logic. If rforge is
missing or incompatible, follow `research-forge` installation guidance and stop.
