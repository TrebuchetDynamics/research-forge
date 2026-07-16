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
	ambiguityOut := filepath.Join(project, "ambiguity.json")
	reportOut := filepath.Join(project, "adjudicated-references.json")
	matchesPath := filepath.Join(project, "matches.json")
	writeJSONFixture(t, matchesPath, parsing.ReferenceNormalizationReport{PaperID: "paper-1", Matches: []parsing.ReferenceMatch{{Index: 0, Ambiguous: true, AmbiguityReason: "low_confidence", Source: "crossref"}}})
	code = Execute([]string{"--json", "--project", project, "parse", "adjudicate-ref", "--parsed", parsed, "--index", "0", "--decision", "defer", "--reviewer", "reviewer-a", "--reason", "needs source check"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("defer code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	code = Execute([]string{"--json", "--project", project, "parse", "adjudicated-refs", "--parsed", parsed, "--matches", matchesPath, "--ambiguity-out", ambiguityOut, "--out", reportOut}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("ambiguity code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if data, err := os.ReadFile(ambiguityOut); err != nil || !bytes.Contains(data, []byte("needs source check")) || !bytes.Contains(data, []byte("crossref")) {
		t.Fatalf("ambiguity data=%s err=%v", string(data), err)
	}
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("log missing: %v", err)
	}
	var writtenReport parsing.ReferenceAdjudicationReport
	if err := readJSONFile(reportOut, &writtenReport); err != nil {
		t.Fatalf("read adjudication report: %v", err)
	}
	if writtenReport.Deferred != 1 || len(writtenReport.Items) != 1 {
		t.Fatalf("written report = %#v", writtenReport)
	}
}

func TestExecuteAdjudicatedRefsDoesNotPartiallyReplaceOutputs(t *testing.T) {
	project := t.TempDir()
	parsed := filepath.Join(project, "parsed.json")
	matchesPath := filepath.Join(project, "matches.json")
	ambiguityOut := filepath.Join(project, "ambiguity.json")
	reportOut := filepath.Join(project, "report-as-directory")
	writeParsedFixture(t, parsed, parsing.ParsedDocument{
		SchemaVersion: "1",
		PaperID:       "paper-1",
		References:    []parsing.Reference{{Title: "Original"}},
	})
	writeJSONFixture(t, matchesPath, parsing.ReferenceNormalizationReport{
		PaperID: "paper-1",
		Matches: []parsing.ReferenceMatch{{Index: 0, Ambiguous: true, AmbiguityReason: "low_confidence", Source: "crossref"}},
	})
	previous := []byte("{\"sentinel\":\"prior ambiguity queue\"}\n")
	if err := os.WriteFile(ambiguityOut, previous, 0o600); err != nil {
		t.Fatalf("write prior ambiguity queue: %v", err)
	}
	if err := os.Mkdir(reportOut, 0o755); err != nil {
		t.Fatalf("create invalid report target: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := Execute([]string{
		"--json", "--project", project, "parse", "adjudicated-refs",
		"--parsed", parsed, "--matches", matchesPath,
		"--ambiguity-out", ambiguityOut, "--out", reportOut,
	}, &stdout, &stderr)
	if code != 1 || !bytes.Contains(stdout.Bytes(), []byte("parse_ref_outputs_write_failed")) {
		t.Fatalf("adjudicated-refs code=%d stderr=%s stdout=%s, want transactional output failure", code, stderr.String(), stdout.String())
	}
	got, err := os.ReadFile(ambiguityOut)
	if err != nil {
		t.Fatalf("read ambiguity queue after failure: %v", err)
	}
	if !bytes.Equal(got, previous) {
		t.Fatalf("ambiguity queue changed after report failure:\n got: %s\nwant: %s", got, previous)
	}
	info, err := os.Stat(ambiguityOut)
	if err != nil {
		t.Fatalf("stat ambiguity queue after failure: %v", err)
	}
	if gotMode := info.Mode().Perm(); gotMode != 0o600 {
		t.Fatalf("ambiguity queue mode = %o, want 600", gotMode)
	}
}
