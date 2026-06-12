# Quickstart

```sh
rforge project create demo --title "Artificial Photosynthesis Review"
rforge search --source openalex --query "artificial photosynthesis"
rforge import json papers.json --project demo
rforge duplicate report --project demo
rforge screen configure --project demo --reason "wrong population"
rforge report build --project demo --out report.md
```
