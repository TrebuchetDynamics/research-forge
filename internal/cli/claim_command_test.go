package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/claims"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

// Missing dispatch, destructive writes, or stale reads must break this workflow.
func TestClaimCommandsAppendAndReadSnapshots(t *testing.T) {
	project := t.TempDir()
	input := filepath.Join(t.TempDir(), "claim.json")
	var stdout, stderr bytes.Buffer
	run := func(args ...string) {
		t.Helper()
		stdout.Reset()
		stderr.Reset()
		if code := Execute(append([]string{"--json", "--project", project, "claim"}, args...), &stdout, &stderr); code != 0 {
			t.Fatalf("%v: code=%d stdout=%s stderr=%s", args, code, &stdout, &stderr)
		}
	}
	run("list")
	if !strings.Contains(stdout.String(), `"claims":[]`) {
		t.Fatalf("new store should be empty: %s", &stdout)
	}
	var first []byte
	for _, text := range []string{"Original assertion.", "Revised assertion."} {
		payload := `{"schemaVersion":"0","id":"c1","text":"` + text + `","evidence":[],"facets":{"semanticSupport":{"state":"passed","checker":{"kind":"agent","id":"pi","model":"model-1"},"timestamp":"2026-10-19T12:00:00Z"},"humanApproval":{"state":"passed","checker":{"kind":"human","id":"ada"},"timestamp":"2026-10-19T12:00:00Z"}}}`
		if err := os.WriteFile(input, []byte(payload), 0o600); err != nil {
			t.Fatal(err)
		}
		run("add", "--file", input)
		data, err := os.ReadFile(filepath.Join(project, "data", "claims.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		if first == nil {
			first = data
		} else if !bytes.HasPrefix(data, first) || bytes.Count(data, []byte{'\n'}) != 2 {
			t.Fatalf("append changed prior snapshot: %s", data)
		}
	}
	run("list")
	var listed struct {
		Data struct {
			Claims []claims.Claim `json:"claims"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Data.Claims) != 1 || listed.Data.Claims[0].Text != "Revised assertion." {
		t.Fatalf("list must select latest snapshot: %s", &stdout)
	}
	run("show", "--id", "c1")
	var shown struct {
		Data struct {
			Claim struct {
				claims.Claim
				SemanticSupportLevel string `json:"semanticSupportLevel"`
			} `json:"claim"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &shown); err != nil {
		t.Fatal(err)
	}
	if shown.Data.Claim.Text != "Revised assertion." || shown.Data.Claim.SemanticSupportLevel != "agent_reported" || shown.Data.Claim.Facets.SourceIdentity != nil || shown.Data.Claim.Facets.HumanApproval.Assessment().Checker.ID != "ada" {
		t.Fatalf("show lost attribution or invented a check: %s", &stdout)
	}
	stdout.Reset()
	if code := Execute([]string{"--json", "--project", project, "claim", "show", "--id", "missing"}, &stdout, &stderr); code == 0 || !strings.Contains(stdout.String(), `"code":"claim_not_found"`) {
		t.Fatalf("missing ID: code=%d stdout=%s", code, &stdout)
	}
}

// Check must append computed facets without rewriting the submitted snapshot.
func TestClaimCheckPersistsSeparateFacets(t *testing.T) {
	project := t.TempDir()
	for _, dir := range []string{"parsed", "data"} {
		if err := os.MkdirAll(filepath.Join(project, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	doc := parsing.EnrichParsedDocumentModel(parsing.ParsedDocument{PaperID: "10.1/test", Title: "Study", ParserName: "fixture", ParserVersion: "1", Sections: []parsing.Section{{Passages: []parsing.Passage{{ID: "p1", Text: "Observed result."}}}}})
	writeJSONForCLITest(t, filepath.Join(project, "parsed", "paper.json"), doc)
	writeJSONForCLITest(t, filepath.Join(project, "data", "library.json"), []library.PaperRecord{{Title: "Study", Identifiers: library.Identifiers{DOI: "10.1/test"}}})
	// The library fixture is legacy (no RecordID); check must retain canonical
	// library loading, which assigns this stable local identity before matching.
	record := claims.Claim{SchemaVersion: claims.SchemaVersion, ID: "c1", Text: "Claim", Evidence: []claims.EvidenceReference{{Identifier: claims.Identifier{Scheme: "local", Value: "rec_d94aa7ead978c2c5aac46af4b57825ab"}, PassageID: "p1", ExactText: "Invented result.", ContentHash: doc.Sections[0].Passages[0].ContentHash, Relation: claims.RelationSupports}}}
	if err := claims.Append(project, record); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(project, claims.RelativePath))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "claim", "check", "--id", "c1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("check code=%d stdout=%s stderr=%s", code, &stdout, &stderr)
	}
	var result struct {
		Data claims.CheckResult `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	f := result.Data.Claim.Facets
	if f.SourceIdentity == nil || f.SourceIdentity.State != claims.StatePassed || f.ContentIntegrity.State != claims.StatePassed || f.PassageMatch.State != claims.StateFailed || f.SemanticSupport != nil || f.HumanApproval != nil {
		t.Fatalf("check = %s", &stdout)
	}
	stored, err := claims.Show(project, "c1")
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(project, claims.RelativePath))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Facets.PassageMatch.State != claims.StateFailed || !bytes.HasPrefix(after, before) || bytes.Count(after, []byte{'\n'}) != 2 {
		t.Fatalf("ledger = %s", after)
	}
}

// Input must pass guarded decoding before any claim can reach the ledger.
func TestClaimAddRejectsMisattributedFacets(t *testing.T) {
	for _, tc := range []struct{ name, facet string }{
		{"engine semantic support", `"semanticSupport":{"state":"passed","checker":{"kind":"engine","id":"rforge","model":"m1"},"timestamp":"2026-10-19T12:00:00Z"}`},
		{"missing agent", `"semanticSupport":{"state":"passed","checker":{"kind":"agent","model":"m1"},"timestamp":"2026-10-19T12:00:00Z"}`},
		{"missing model", `"semanticSupport":{"state":"passed","checker":{"kind":"agent","id":"pi"},"timestamp":"2026-10-19T12:00:00Z"}`},
		{"unnamed reviewer", `"humanApproval":{"state":"passed","checker":{"kind":"human","id":" "},"timestamp":"2026-10-19T12:00:00Z"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project := t.TempDir()
			input := filepath.Join(t.TempDir(), "claim.json")
			payload := `{"schemaVersion":"0","id":"c1","text":"Assertion.","evidence":[],"facets":{` + tc.facet + `}}`
			if err := os.WriteFile(input, []byte(payload), 0o600); err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			code := Execute([]string{"--json", "--project", project, "claim", "add", "--file", input}, &stdout, &stderr)
			if code == 0 || !strings.Contains(stdout.String(), `"code":"claim_input_invalid"`) {
				t.Fatalf("accepted invalid facet: code=%d stdout=%s stderr=%s", code, &stdout, &stderr)
			}
			if _, err := os.Stat(filepath.Join(project, "data", "claims.jsonl")); !os.IsNotExist(err) {
				t.Fatalf("invalid claim touched ledger: %v", err)
			}
		})
	}
}
