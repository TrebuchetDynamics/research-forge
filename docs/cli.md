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
- `rforge ui` reports the local Go + HTMX web GUI status; `rforge --json ui` exposes the selected stack and ready state from ADR 0006
- `rforge decisions` lists owner decisions and implementation trackers that intentionally keep remaining TODO items open
- `rforge decisions --check TODO.md` verifies unchecked TODO items, line references, and tracking issue references are decision/tracker-covered
- `rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md` verifies decision/tracker coverage plus the closeout prompt-to-artifact audit; JSON output includes `completion_blocked`, `blocked_decisions`, `blocked_decision_ids`, and `license_resolution_verified` so automation does not treat covered TODOs as finished work
- `rforge --json decisions` exposes blocker routing and choice metadata such as `issue_title`, `todo_refs`, `issue_labels`, `milestone`, `options_considered`, and `owner_response_required_fields` for owner-decision automation
- `rforge decisions --markdown` prints a review-friendly decision/evidence table with routing/options metadata
- `rforge decisions --issue-body <decision-id>` prints an owner-decision issue body scaffold
- `make license-decision-live-audit` inspects live issue #1 approval fields; `make license-decision-approval-gate` passes only when the aggregate is `approved:true`
