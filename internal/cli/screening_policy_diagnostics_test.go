package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/screening"
)

func TestExecuteScreenSensitivityWritesBalancedPolicyDiagnostics(t *testing.T) {
	project := t.TempDir()
	if code := Execute([]string{"project", "create", project, "--title", "Screen Sensitivity"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code = %d", code)
	}
	if code := Execute([]string{"--project", project, "screen", "configure", "--reason", "off-topic"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("configure code = %d", code)
	}
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	inputs := []library.PaperRecordInput{{Title: "Solar catalyst", Identifiers: library.Identifiers{DOI: "10.1/in"}}, {Title: "Battery", Identifiers: library.Identifiers{DOI: "10.1/out"}}, {Title: "Solar fuel", Identifiers: library.Identifiers{DOI: "10.1/candidate"}}}
	papers := []library.PaperRecord{}
	for _, input := range inputs {
		p, _ := library.NewPaperRecord(input)
		papers = append(papers, p)
	}
	if _, err := store.ImportRecords(papers); err != nil {
		t.Fatalf("import: %v", err)
	}
	mustScreen(t, project, "10.1/in", "include", "", "r1")
	mustScreen(t, project, "10.1/out", "exclude", "off-topic", "r1")
	out := filepath.Join(project, "data", "screening-sensitivity.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "screen", "sensitivity", "--stage", "title_abstract", "--relevant", "10.1/candidate", "--target-recall", "1.0", "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var report screening.ActiveLearningSensitivityReport
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if report.SelectedPolicy == "" || len(report.PolicyResults) == 0 || report.PolicyResults[0].Simulation.TotalRelevant != 1 {
		t.Fatalf("report = %#v", report)
	}
}
