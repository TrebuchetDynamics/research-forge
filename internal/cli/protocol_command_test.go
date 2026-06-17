package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestExecuteProtocolCompileJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "protocol", "compile", "--type", "pico", "--question", "Do catalysts improve hydrogen evolution?", "--population", "hydrogen evolution", "--intervention", "catalysts", "--comparator", "baseline", "--outcome", "efficiency"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Plan struct {
				Framework                string `json:"framework"`
				ReviewerApprovalRequired bool   `json:"reviewerApprovalRequired"`
				AutoAcceptedClaims       bool   `json:"autoAcceptedClaims"`
				SourceQueries            map[string]struct{ Query string `json:"query"` } `json:"sourceQueries"`
				ExtractionSchema         struct{ Fields []struct{ Name string `json:"name"` } `json:"fields"` } `json:"extractionSchema"`
			} `json:"plan"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json decode: %v\n%s", err, stdout.String())
	}
	if !env.OK || env.Data.Plan.Framework != "pico" || !env.Data.Plan.ReviewerApprovalRequired || env.Data.Plan.AutoAcceptedClaims {
		t.Fatalf("unexpected plan gates: %#v", env.Data.Plan)
	}
	if env.Data.Plan.SourceQueries["openalex"].Query == "" || env.Data.Plan.SourceQueries["semantic-scholar"].Query == "" {
		t.Fatalf("missing source queries: %#v", env.Data.Plan.SourceQueries)
	}
	if !strings.Contains(stdout.String(), "support_ref") {
		t.Fatalf("schema missing support_ref: %s", stdout.String())
	}
}

func TestExecuteProtocolCompileRequiresQuestion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"protocol", "compile", "--type", "pico"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "research question is required") {
		t.Fatalf("stderr missing error: %s", stderr.String())
	}
}
