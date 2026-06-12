# Example open-data project

This synthetic project demonstrates an artificial-photosynthesis review without copyrighted PDFs or private data.

Contents to generate locally:

```sh
rforge project create examples/open-data-project/demo --title "Artificial Photosynthesis Open Data Review"
rforge import json examples/open-data-project/papers.json --project examples/open-data-project/demo
rforge duplicate report --project examples/open-data-project/demo
rforge report build --project examples/open-data-project/demo --out examples/open-data-project/report.md
```

All records are deterministic fixtures for testing ResearchForge workflows.
