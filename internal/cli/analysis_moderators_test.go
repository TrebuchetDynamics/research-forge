package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/analysis"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

func TestExecuteAnalysisSubgroupAndMetaRegressionFromEvidenceModerators(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Moderators"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code = %d", code)
	}
	run := analysis.AnalysisRun{SchemaVersion: "1", ID: "run1", InputRows: []analysis.InputRow{{PaperID: "p1", EffectSize: 1, Variance: 1}, {PaperID: "p2", EffectSize: 2, Variance: 1}}}
	if err := os.MkdirAll(filepath.Join(project, "analysis"), 0o755); err != nil {
		t.Fatalf("mkdir analysis: %v", err)
	}
	writeJSONForCLITest(t, filepath.Join(project, "analysis", "run1.json"), run)
	items := []evidence.EvidenceItem{
		{PaperID: "p1", Status: evidence.StatusAccepted, Values: map[string]string{"region": "EU", "dose": "1"}},
		{PaperID: "p2", Status: evidence.StatusAccepted, Values: map[string]string{"region": "US", "dose": "2"}},
	}
	writeJSONForCLITest(t, filepath.Join(project, "data", "evidence.items.json"), items)
	var stdout, stderr bytes.Buffer
	if code := Execute([]string{"--json", "--project", project, "analysis", "moderators", "run1"}, &stdout, &stderr); code != 0 {
		t.Fatalf("moderators code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "region") || !strings.Contains(stdout.String(), "dose") {
		t.Fatalf("moderators output = %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", project, "analysis", "subgroup", "run1", "--variable", "region", "--from-evidence", "region"}, &stdout, &stderr); code != 0 {
		t.Fatalf("subgroup code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"variable":"region"`) {
		t.Fatalf("subgroup output = %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "--project", project, "analysis", "meta-regression", "run1", "--moderator", "dose", "--from-evidence", "dose"}, &stdout, &stderr); code != 0 {
		t.Fatalf("meta-regression code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"moderator":"dose"`) {
		t.Fatalf("meta-regression output = %s", stdout.String())
	}
}
