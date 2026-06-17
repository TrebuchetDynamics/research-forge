package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
)

func TestExecuteEvidenceCitationLockedSuggestAndReview(t *testing.T) {
	project := t.TempDir()
	if code := Execute([]string{"project", "create", project, "--title", "Citation Locked"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code = %d", code)
	}
	out := filepath.Join(project, "data", "citation-locked.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "evidence", "citation-suggest", "--paper", "paper-1", "--kind", "report_prose", "--prompt", "summarize outcome", "--support", "paper-1:p1=Mortality was lower in the treatment group.", "--model", "fixture-llm", "--version", "1.0", "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("suggest code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var queue evidence.CitationLockedSuggestionQueue
	if err := readJSONFile(out, &queue); err != nil {
		t.Fatalf("read queue: %v", err)
	}
	if len(queue.Suggestions) != 1 || queue.Suggestions[0].Status != evidence.StatusSuggested || len(queue.Suggestions[0].CitationLocks) != 1 {
		t.Fatalf("queue = %#v", queue)
	}
	reviewOut := filepath.Join(project, "data", "citation-locked-reviewed.json")
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "evidence", "citation-review", "--queue", out, "--id", queue.Suggestions[0].ID, "--decision", "accepted", "--reviewer", "ada", "--note", "checked citation", "--out", reviewOut}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("review code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var reviewed evidence.CitationLockedSuggestionQueue
	if err := readJSONFile(reviewOut, &reviewed); err != nil {
		t.Fatalf("read reviewed: %v", err)
	}
	if reviewed.Suggestions[0].Status != evidence.StatusAccepted || reviewed.Suggestions[0].ReviewerDecision.Reviewer != "ada" {
		t.Fatalf("reviewed = %#v", reviewed)
	}

	queryOut := filepath.Join(project, "data", "citation-locked-query.json")
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "evidence", "citation-suggest", "--paper", "paper-1", "--kind", "query_expansion", "--prompt", "suggest expansion", "--support", "paper-1:p2=The trial studied hospitalization outcomes.", "--out", queryOut}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("query suggest code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var queryQueue evidence.CitationLockedSuggestionQueue
	if err := readJSONFile(queryOut, &queryQueue); err != nil {
		t.Fatalf("read query queue: %v", err)
	}
	if queryQueue.Suggestions[0].Kind != evidence.CitationLockedQueryExpansion || !evidence.EverySuggestedSentenceCitationLocked(queryQueue.Suggestions[0]) {
		t.Fatalf("query queue = %#v", queryQueue)
	}
}
