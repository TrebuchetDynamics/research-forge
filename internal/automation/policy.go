package automation

import "strings"

// Actor names who may make a workflow decision under the ResearchForge
// retrieval-first, provenance-first automation policy.
type Actor string

const (
	ActorAgent Actor = "agent"
	ActorHuman Actor = "human"
)

// DecisionClass classifies whether an LLM/automation agent may execute a step
// or must stop at a human gate.
type DecisionClass string

const (
	DecisionAgentAllowed  DecisionClass = "agent_allowed"
	DecisionHumanRequired DecisionClass = "human_required"
)

// PolicyDecision is the machine-readable contract agents use before acting.
type PolicyDecision struct {
	Action            string        `json:"action"`
	Class             DecisionClass `json:"class"`
	AllowedActor      Actor         `json:"allowedActor"`
	RequiresHuman     bool          `json:"requiresHuman"`
	Gate              string        `json:"gate,omitempty"`
	AuditArtifact     string        `json:"auditArtifact"`
	Reason            string        `json:"reason"`
	LLMAgentMayRun    bool          `json:"llmAgentMayRun"`
	LLMAgentMayDecide bool          `json:"llmAgentMayDecide"`
}

// DefaultPolicy returns the hybrid automation policy: agents may decide and run
// reversible assistive steps, but humans approve scientific, legal/privacy, and
// final-claim decisions.
func DefaultPolicy() []PolicyDecision {
	return []PolicyDecision{
		agent("batch-retrieval-dedupe", "results-deduped.jsonl", "agents may run multi-source retrieval, normalize records, dedupe DOI/title matches, and log failures because no scientific inclusion decision is made"),
		agent("screening-queue", "data/screening-audit-bundle.json", "agents may rank, prioritize, assign, and surface uncertainty queues; include/exclude decisions remain gated"),
		agent("extraction-suggest", "data/evidence-grid.json", "agents may create schema-constrained extraction suggestions only when exact support refs are attached"),
		agent("analysis-prepare", "analysis/*-manifest.json", "agents may prepare deterministic analysis inputs and scripts from accepted evidence without changing accepted scientific facts"),
		agent("claim-trace-audit", "data/claim-panel.json", "agents may audit report claims and block export on weak or unresolved support"),
		human("screening-decision", "screening approval", "data/screening.events.json", "include, exclude, uncertain, and adjudication decisions determine study eligibility and require a human reviewer"),
		human("full-text-acquisition-approval", "full-text acquisition approval", "data/legal-acquisition-queue.json", "downloading or archiving full text can create copyright/privacy obligations and requires human approval"),
		human("extraction-acceptance", "evidence extraction approval", "data/evidence.items.json", "accepted evidence feeds meta-analysis and report claims, so suggestions require reviewer acceptance/correction/rejection"),
		human("analysis-method-selection", "analysis method approval", "analysis/method-workbench.json", "selecting statistical methods can change scientific conclusions and requires reviewer rationale"),
		human("final-claim-approval", "claim approval", "data/claim-panel.json", "final report claims require human approval after traceability checks pass"),
		human("package-export-approval", "package approval", "review.rforgepkg", "shareable packages may include copyrighted, private, or reviewer-sensitive artifacts and require approval"),
	}
}

// Evaluate returns the policy decision for an action. Unknown actions are human
// gated by default so agents fail closed.
func Evaluate(action string) PolicyDecision {
	action = strings.TrimSpace(action)
	for _, decision := range DefaultPolicy() {
		if decision.Action == action {
			return decision
		}
	}
	return PolicyDecision{Action: action, Class: DecisionHumanRequired, AllowedActor: ActorHuman, RequiresHuman: true, Gate: "owner approval", AuditArtifact: "provenance log", Reason: "unknown automation action; fail closed until an owner defines the gate", LLMAgentMayRun: false, LLMAgentMayDecide: false}
}

func agent(action, artifact, reason string) PolicyDecision {
	return PolicyDecision{Action: action, Class: DecisionAgentAllowed, AllowedActor: ActorAgent, RequiresHuman: false, AuditArtifact: artifact, Reason: reason, LLMAgentMayRun: true, LLMAgentMayDecide: true}
}

func human(action, gate, artifact, reason string) PolicyDecision {
	return PolicyDecision{Action: action, Class: DecisionHumanRequired, AllowedActor: ActorHuman, RequiresHuman: true, Gate: gate, AuditArtifact: artifact, Reason: reason, LLMAgentMayRun: false, LLMAgentMayDecide: false}
}
