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

func TestExecuteScreenAssignPanelAndAuditBundle(t *testing.T) {
	project := t.TempDir()
	if code := Execute([]string{"project", "create", project, "--title", "Screen Audit"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code = %d", code)
	}
	if code := Execute([]string{"--project", project, "screen", "configure", "--reason", "off-topic"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("configure code = %d", code)
	}
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	papers := []library.PaperRecord{}
	for _, input := range []library.PaperRecordInput{{Title: "Paper 1", Identifiers: library.Identifiers{DOI: "p1"}}, {Title: "Paper 2", Identifiers: library.Identifiers{DOI: "p2"}}} {
		paper, _ := library.NewPaperRecord(input)
		papers = append(papers, paper)
	}
	if _, err := store.ImportRecords(papers); err != nil {
		t.Fatalf("import: %v", err)
	}
	assignOut := filepath.Join(project, "data", "assignments.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "screen", "assign", "--stage", "title_abstract", "--reviewer", "ada", "--reviewer", "bob", "--per-record", "2", "--out", assignOut}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("assign code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var assignments []screening.ReviewerAssignment
	if err := readJSONFile(assignOut, &assignments); err != nil {
		t.Fatalf("read assignments: %v", err)
	}
	if len(assignments) != 4 {
		t.Fatalf("assignments = %#v", assignments)
	}
	mustScreen(t, project, "p1", "include", "", "ada")
	mustScreen(t, project, "p1", "exclude", "off-topic", "bob")
	mustScreen(t, project, "p2", "uncertain", "", "ada")
	panelOut := filepath.Join(project, "data", "panel.json")
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "screen", "panel", "--stage", "title_abstract", "--out", panelOut}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("panel code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var panel screening.ConflictAdjudicationPanel
	if err := readJSONFile(panelOut, &panel); err != nil {
		t.Fatalf("read panel: %v", err)
	}
	if len(panel.Conflicts) != 1 || panel.Conflicts[0].PaperID != "p1" {
		t.Fatalf("panel = %#v", panel)
	}
	bundleOut := filepath.Join(project, "data", "screening-audit-bundle.json")
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "screen", "audit-bundle", "--stage", "title_abstract", "--assignments", assignOut, "--out", bundleOut}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("bundle code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var bundle screening.ScreeningAuditBundle
	data, err := os.ReadFile(bundleOut)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	if bundle.InputHash == "" || len(bundle.Assignments) != 4 || len(bundle.Panel.Conflicts) != 1 || len(bundle.Uncertain) != 1 {
		t.Fatalf("bundle = %#v", bundle)
	}
}
