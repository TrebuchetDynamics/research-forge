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

- Calibration/evaluation reports for suggestion quality.
- Dashboard panels for unresolved extraction/risk-of-bias suggestions.

Implemented:

- `DefaultRiskOfBiasSchemaTemplates`, `DraftRiskOfBiasSuggestionQueue`, `EveryRiskOfBiasJudgmentAuditable`, and `rforge evidence risk-bias-*` implement RobotReviewer-inspired risk-of-bias/evidence-suggestion workflows where every automated judgment carries exact support text/ref, uncertainty, model/version metadata, and accept/correct/reject reviewer state.

## Recommended next slice

Add calibration/evaluation reporting and dashboard panels for unresolved RobotReviewer-inspired extraction/risk-of-bias suggestions.
