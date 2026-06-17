# RobotReviewer study note

- Repository/ecosystem: `ijmarshall/robotreviewer` and related risk-of-bias automation research.
- Area: evidence extraction, randomized-trial risk of bias, citation-linked review assistance.
- Disposition: `pattern-reference`.
- License/action constraint: study workflow concepts only; do not port models, biomedical training data, or generated templates without license and validation review.

## Why it matters

RobotReviewer shows how automated extraction and risk-of-bias assistance can speed evidence review while still requiring human judgment. ResearchForge needs this pattern for citation-locked suggestions that never become accepted evidence without reviewer approval.

## Patterns to learn

- Suggested judgments should link to exact supporting text spans.
- Risk-of-bias domains are schema-driven and reviewer-audited.
- Automation must expose uncertainty and rationale, not only labels.
- Biomedical-focused features should be generalized carefully for non-RCT scientific domains.

## ResearchForge status

Implemented nearby capabilities:

- Evidence schemas and manual evidence entry.
- LLM suggestion adapter interface with accepted/rejected/corrected transitions.
- Source-support requirements for accepted evidence.
- Report audit and redaction paths.

Missing features:

- Risk-of-bias schema templates with domain-specific fields.
- Suggestion cards with quoted support, uncertainty, and reviewer decision state.
- Model/version/provenance records for automated extraction suggestions.
- Calibration/evaluation reports for suggestion quality.
- Dashboard panels for unresolved extraction/risk-of-bias suggestions.

## Recommended next slice

Add a risk-of-bias extraction schema template and an evidence-suggestion review queue where every suggested judgment cites a passage and remains unaccepted until reviewer action.
