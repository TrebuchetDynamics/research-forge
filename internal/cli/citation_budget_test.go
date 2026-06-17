package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExecuteCitationsExpandDryRunBudgetEstimate(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "citations", "expand", "--source", "openalex", "--paper", "W1", "--direction", "both", "--out", "graph.json", "--depth", "3", "--max-records", "50", "--max-api-calls", "7", "--retry-budget", "2", "--resume-cursor", "frontier:W2", "--dry-run"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	for _, want := range []string{"budgetEstimate", "maxApiCalls", "frontier:W2", "dryRunPlan"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("missing %s in %s", want, stdout.String())
		}
	}
}
