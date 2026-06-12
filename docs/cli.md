# CLI reference

Core commands:

- `rforge project create [path] --title <title>`
- `rforge project inspect <path>`
- `rforge search --source openalex|arxiv|crossref --query <query>`
- `rforge oa lookup <doi>`
- `rforge import json|csv|bibtex|ris <file>`
- `rforge export json|csv|bibtex|ris <file>`
- `rforge duplicate report|merge|split`
- `rforge oss add|list|clone|license-check|note|scan|report|refresh`
- `rforge pdf fetch --doi <doi>`
- `rforge parse --paper <id> --parser grobid --pdf <file>`
- `rforge index rebuild`, `rforge retrieve --query <query>`
- `rforge screen configure|decide|queue`, `rforge prisma counts`
- `rforge extraction schema add`, `rforge extract add|suggest`, `rforge evidence audit`
- `rforge analysis prepare|run|export`
- `rforge report build|audit`
- `rforge decisions` lists owner/build decisions that intentionally block remaining TODO items
- `rforge decisions --check TODO.md` verifies unchecked TODO items and line references are decision-covered
- `rforge decisions --markdown` prints a review-friendly decision/evidence table
- `rforge decisions --issue-body <decision-id>` prints an owner-decision issue body scaffold
