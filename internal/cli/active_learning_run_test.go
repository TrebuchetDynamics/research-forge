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

func TestExecuteScreenActiveRunPersistsASReviewStyleRun(t *testing.T) {
	project := t.TempDir()
	if code := Execute([]string{"project", "create", project, "--title", "Screening"}, ioDiscard{}, ioDiscard{}); code != 0 {
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
	for _, input := range []library.PaperRecordInput{{Title: "Solar catalyst", Abstract: "water splitting", Identifiers: library.Identifiers{DOI: "10.1/in"}}, {Title: "Battery storage", Abstract: "excluded", Identifiers: library.Identifiers{DOI: "10.1/out"}}, {Title: "Solar fuel candidate", Abstract: "catalyst", Identifiers: library.Identifiers{DOI: "10.1/cand"}}} {
		paper, err := library.NewPaperRecord(input)
		if err != nil {
			t.Fatalf("paper: %v", err)
		}
		papers = append(papers, paper)
	}
	if _, err := store.ImportRecords(papers); err != nil {
		t.Fatalf("import papers: %v", err)
	}
	mustScreen(t, project, "10.1/in", "include", "", "r1")
	mustScreen(t, project, "10.1/out", "exclude", "off-topic", "r2")
	out := filepath.Join(project, "data", "active-learning-runs", "run.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "screen", "active-run", "--stage", "title_abstract", "--method", "active-learning", "--target-recall", "0.8", "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("active-run code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var run screening.ActiveLearningRun
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read run: %v", err)
	}
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	if run.InputHash == "" || run.DecisionHash == "" || run.RankingMethod != "active-learning" || len(run.SeedDecisions) != 2 || len(run.RankedOutput) != 1 || run.StoppingDiagnostics.TargetRecall != 0.8 {
		t.Fatalf("run = %#v", run)
	}
}

func mustScreen(t *testing.T, project, paper, decision, reason, reviewer string) {
	t.Helper()
	args := []string{"--project", project, "screen", "decide", "--paper", paper, "--stage", "title_abstract", "--decision", decision, "--reviewer", reviewer}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	if code := Execute(args, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("screen %s code = %d", paper, code)
	}
}
