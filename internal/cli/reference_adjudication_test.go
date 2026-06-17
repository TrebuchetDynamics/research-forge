package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestExecuteParseAdjudicateRefPersistsAndReportsDecisions(t *testing.T) {
	project := t.TempDir()
	parsed := filepath.Join(project, "parsed.json")
	writeParsedFixture(t, parsed, parsing.ParsedDocument{SchemaVersion: "1", PaperID: "paper-1", References: []parsing.Reference{{Title: "Original"}}})
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "parse", "adjudicate-ref", "--parsed", parsed, "--index", "0", "--decision", "correct", "--reviewer", "reviewer-a", "--reason", "fix title", "--title", "Corrected", "--doi", "10.1/c"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("adjudicate code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	logPath := filepath.Join(project, "data", "reference-adjudications.jsonl")
	records, err := parsing.LoadReferenceAdjudications(logPath)
	if err != nil || len(records) != 1 || records[0].Decision != "correct" {
		t.Fatalf("records=%#v err=%v", records, err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "parse", "adjudicated-refs", "--parsed", parsed}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("report code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var env struct {
		Data struct {
			Report parsing.ReferenceAdjudicationReport `json:"referenceAdjudication"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v\n%s", err, stdout.String())
	}
	if env.Data.Report.Corrected != 1 || env.Data.Report.Items[0].Reference.Title != "Corrected" {
		t.Fatalf("report = %#v", env.Data.Report)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("log missing: %v", err)
	}
}
