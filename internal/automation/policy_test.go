package automation

import "testing"

func TestDefaultPolicyAllowsAgentsOnlyForAssistiveReversibleSteps(t *testing.T) {
	for _, action := range []string{"batch-retrieval-dedupe", "screening-queue", "extraction-suggest", "analysis-prepare", "claim-trace-audit"} {
		decision := Evaluate(action)
		if decision.Class != DecisionAgentAllowed || decision.RequiresHuman || !decision.LLMAgentMayDecide || decision.AllowedActor != ActorAgent {
			t.Fatalf("%s decision = %#v, want agent allowed", action, decision)
		}
		if decision.AuditArtifact == "" || decision.Reason == "" {
			t.Fatalf("%s missing audit artifact/reason: %#v", action, decision)
		}
	}
}

func TestDefaultPolicyRequiresHumansForScientificLegalAndFinalClaimGates(t *testing.T) {
	for _, action := range []string{"screening-decision", "full-text-acquisition-approval", "extraction-acceptance", "analysis-method-selection", "final-claim-approval", "package-export-approval"} {
		decision := Evaluate(action)
		if decision.Class != DecisionHumanRequired || !decision.RequiresHuman || decision.LLMAgentMayDecide || decision.AllowedActor != ActorHuman {
			t.Fatalf("%s decision = %#v, want human gate", action, decision)
		}
		if decision.Gate == "" || decision.AuditArtifact == "" {
			t.Fatalf("%s missing gate/audit artifact: %#v", action, decision)
		}
	}
}

func TestUnknownAutomationActionFailsClosed(t *testing.T) {
	decision := Evaluate("invent-new-conclusion")
	if decision.Class != DecisionHumanRequired || !decision.RequiresHuman || decision.Gate != "owner approval" {
		t.Fatalf("unknown action should fail closed: %#v", decision)
	}
}
