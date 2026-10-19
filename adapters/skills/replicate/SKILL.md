---
name: replicate
description: Use only when explicitly asked to run the ResearchForge replicate workflow.
disable-model-invocation: true
---

# Replication plan

Load the `research-forge` methodology skill first, including its engine compatibility
check. It owns research decisions, provenance, verification and human gates.

For the requested paper and plan output file, make this engine call:

```sh
rforge replicate <doi-or-arxiv-id> --out <plan.md>
```

This requests a plan, not permission to execute experiments. Resolve missing inputs
with the user and preserve their scope and restrictions. Use the engine's results;
this adapter supplies no research logic. If rforge is missing or incompatible,
follow `research-forge` installation guidance and stop.
