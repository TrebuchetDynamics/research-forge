package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/claims"
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
