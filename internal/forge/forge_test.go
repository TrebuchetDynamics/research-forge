package forge

import (
	"strings"
	"testing"
)

func TestGuidedWorkflowCapturesSourceToolChoicesAndPrivacyPreview(t *testing.T) {
	projectPath := t.TempDir()
	state, err := Init(projectPath, InitOptions{Question: "Do catalysts improve hydrogen evolution?", SourceChoices: []string{"openalex", "semantic-scholar"}, ToolChoices: []string{"grobid", "qdrant"}, Actor: "tester"})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if len(state.SourceChoices) != 2 || len(state.ToolChoices) != 2 || len(state.PrivacyLegalPreview) == 0 {
		t.Fatalf("state missing guided choices/privacy preview: %#v", state)
	}
	if !strings.Contains(strings.Join(state.PrivacyLegalPreview, " "), "reviewer approval") {
		t.Fatalf("preview missing reviewer approval warning: %#v", state.PrivacyLegalPreview)
	}
}

func TestGuidedWorkflowRequiresReviewGatesBeforeIrreversibleTransitions(t *testing.T) {
	projectPath := t.TempDir()
	state, err := Init(projectPath, InitOptions{Question: "Do catalysts improve hydrogen evolution?", Actor: "tester"})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if state.CurrentState != StateQuestionDraft || !state.BlockedBy("question approval") {
		t.Fatalf("init state = %#v", state)
	}
	if _, err := Next(projectPath, "tester"); err == nil || !strings.Contains(err.Error(), "blocked review gate") {
		t.Fatalf("next before approval err = %v", err)
	}

	state, err = Approve(projectPath, ApprovalInput{Gate: "question approval", Note: "canonical question accepted", Actor: "reviewer"})
	if err != nil {
		t.Fatalf("approve question: %v", err)
	}
	if state.CurrentState != StateProtocolPlan || !state.BlockedBy("protocol approval") {
		t.Fatalf("after question approval = %#v", state)
	}

	state, err = Approve(projectPath, ApprovalInput{Gate: "protocol approval", Note: "criteria acceptable", Actor: "reviewer"})
	if err != nil {
		t.Fatalf("approve protocol: %v", err)
	}
	if state.CurrentState != StateSourcePlan || !state.BlockedBy("network/API approval") {
		t.Fatalf("after protocol approval = %#v", state)
	}

	status, err := Status(projectPath)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(status.NextSafeActions) == 0 || !strings.Contains(status.NextSafeActions[0].CLI, "rforge protocol") {
		t.Fatalf("next actions = %#v", status.NextSafeActions)
	}
	events, err := ProvenanceEvents(projectPath)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	transitions := 0
	for _, event := range events {
		if event.Action == "forge.state.transition" {
			transitions++
		}
	}
	if transitions < 3 {
		t.Fatalf("want provenance transitions, got %d events=%#v", transitions, events)
	}
}

func TestReviewGateCatalogCoversIrreversibleScientificAndSharingDecisions(t *testing.T) {
	gates := ReviewGates()
	want := []string{"protocol approval", "network/API approval", "identity approval", "legal acquisition approval", "parser arbitration approval", "screening approval", "evidence approval", "analysis approval", "claim approval", "package approval"}
	for _, gate := range want {
		if _, ok := gates[gate]; !ok {
			t.Fatalf("missing gate %q in %#v", gate, gates)
		}
	}
}
