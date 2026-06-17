package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

func TestExecuteAnalysisPrepareSupportsAdditionalEffectCalculators(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Effects"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code = %d", code)
	}
	items := []evidence.EvidenceItem{{PaperID: "paper-1", Values: map[string]string{"mean_treatment": "10", "mean_control": "8", "sd_treatment": "4", "sd_control": "5", "n_treatment": "25", "n_control": "25"}, Status: evidence.StatusAccepted}}
	writeJSONForCLITest(t, filepath.Join(project, "data", "evidence.items.json"), items)
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "analysis", "prepare", "run-md", "--effect", "mean-difference"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("prepare code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "paper-1") {
		t.Fatalf("stdout = %s", stdout.String())
	}
}
