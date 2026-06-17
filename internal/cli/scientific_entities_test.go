package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestExecuteEvidenceEntitySuggestAndReview(t *testing.T) {
	project := t.TempDir()
	if code := Execute([]string{"project", "create", project, "--title", "Entities"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code = %d", code)
	}
	parsed := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "paper-1", ParserName: "grobid", ParserVersion: "0.8", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", Text: "Tumor necrosis factor (TNF) increased."}}}}})
	parsedPath := filepath.Join(project, "parsed", "paper-1.json")
	if err := os.MkdirAll(filepath.Dir(parsedPath), 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	writeJSONForCLITest(t, parsedPath, parsed)
	out := filepath.Join(project, "data", "entity-suggestions.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "evidence", "entity-suggest", "--parsed", parsedPath, "--model", "scispacy-fixture", "--version", "1.0", "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("suggest code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var queue evidence.ScientificEntitySuggestionQueue
	if err := readJSONFile(out, &queue); err != nil {
		t.Fatalf("read queue: %v", err)
	}
	if len(queue.Suggestions) == 0 || queue.Suggestions[0].PassageID == "" || len(queue.Suggestions[0].EntityLinkCandidates) == 0 {
		t.Fatalf("queue = %#v", queue)
	}
	reviewOut := filepath.Join(project, "data", "entity-reviewed.json")
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "evidence", "entity-review", "--queue", out, "--id", queue.Suggestions[0].ID, "--decision", "accepted", "--reviewer", "ada", "--out", reviewOut}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("review code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var reviewed evidence.ScientificEntitySuggestionQueue
	if err := readJSONFile(reviewOut, &reviewed); err != nil {
		t.Fatalf("read reviewed: %v", err)
	}
	if reviewed.Suggestions[0].ReviewerDecision.Reviewer != "ada" {
		t.Fatalf("reviewed = %#v", reviewed)
	}
}
