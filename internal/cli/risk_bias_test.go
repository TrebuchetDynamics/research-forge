package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

func TestExecuteRiskBiasTemplatesSuggestAndReview(t *testing.T) {
	project := t.TempDir()
	if code := Execute([]string{"project", "create", project, "--title", "Risk Bias"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code = %d", code)
	}
	templatesOut := filepath.Join(project, "data", "rob-templates.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "evidence", "risk-bias-templates", "--out", templatesOut}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("templates code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var templates []evidence.RiskOfBiasTemplate
	if err := readJSONFile(templatesOut, &templates); err != nil || len(templates) == 0 {
		t.Fatalf("templates read len=%d err=%v", len(templates), err)
	}
	queueOut := filepath.Join(project, "data", "risk-bias-queue.json")
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "evidence", "risk-bias-suggest", "--paper", "paper-1", "--support", "sec1:p1=Random sequence generation was computer generated. Participants were not blinded.", "--model", "robotreviewer-fixture", "--version", "1.0", "--out", queueOut}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("suggest code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var queue evidence.RiskOfBiasSuggestionQueue
	if err := readJSONFile(queueOut, &queue); err != nil || len(queue.Suggestions) == 0 {
		t.Fatalf("queue len=%d err=%v", len(queue.Suggestions), err)
	}
	if queue.Suggestions[0].ExactSupportText == "" || queue.Suggestions[0].ModelVersion != "1.0" {
		t.Fatalf("queue = %#v", queue)
	}
	reviewOut := filepath.Join(project, "data", "risk-bias-reviewed.json")
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "evidence", "risk-bias-review", "--queue", queueOut, "--id", queue.Suggestions[0].ID, "--decision", "accepted", "--reviewer", "ada", "--note", "checked exact quote", "--out", reviewOut}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("review code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var reviewed evidence.RiskOfBiasSuggestionQueue
	data := stdout.Bytes()
	if len(data) == 0 {
		t.Fatalf("empty stdout")
	}
	if err := readJSONFile(reviewOut, &reviewed); err != nil {
		t.Fatalf("reviewed read: %v", err)
	}
	if reviewed.Suggestions[0].ReviewerDecision.Reviewer != "ada" {
		encoded, _ := json.Marshal(reviewed)
		t.Fatalf("reviewed = %s", encoded)
	}
}
