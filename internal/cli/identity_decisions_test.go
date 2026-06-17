package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

func TestExecuteLibraryIdentityConflictsAndDecisionRecord(t *testing.T) {
	project := t.TempDir()
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	left, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Catalyst A", Identifiers: library.Identifiers{DOI: "10.1000/same"}, Year: 2020})
	right, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Unrelated title", Identifiers: library.Identifiers{CrossrefID: "10.1000/same"}, Year: 2024})
	if err := store.ReplaceAll([]library.PaperRecord{left, right}); err != nil {
		t.Fatalf("replace: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "library", "identity-conflicts"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("identity-conflicts code=%d stderr=%s", code, stderr.String())
	}
	var conflictsEnv struct {
		OK   bool `json:"ok"`
		Data struct {
			Conflicts []library.IdentityConflictRecord `json:"conflicts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &conflictsEnv); err != nil {
		t.Fatalf("decode conflicts: %v\n%s", err, stdout.String())
	}
	if !conflictsEnv.OK || len(conflictsEnv.Data.Conflicts) != 1 {
		t.Fatalf("conflicts env = %#v", conflictsEnv)
	}

	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "library", "identity-decision", "record", "--action", "merge", "--cluster", conflictsEnv.Data.Conflicts[0].ClusterID, "--reason", "reviewer accepted DOI match", "--reviewer", "reviewer-a", "--before-indexes", "0,1", "--after-indexes", "0"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("identity-decision code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	log, err := library.ReadIdentityDecisionLog(filepath.Join(project, "data", "identity-decisions.jsonl"))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if len(log.Decisions) != 1 || !log.Decisions[0].Reversible || log.Decisions[0].Action != library.IdentityDecisionMerge || len(log.Decisions[0].Before) != 2 || len(log.Decisions[0].After) != 1 {
		t.Fatalf("decision log = %#v", log)
	}

	stdout.Reset()
	stderr.Reset()
	code = Execute([]string{"--json", "--project", project, "library", "identity-decision", "apply", "--id", log.Decisions[0].ID}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("identity-decision apply code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	papers, err := store.List()
	if err != nil {
		t.Fatalf("list after apply: %v", err)
	}
	if len(papers) != 1 || papers[0].Identifiers.DOI != "10.1000/same" {
		t.Fatalf("papers after apply = %#v", papers)
	}
}
