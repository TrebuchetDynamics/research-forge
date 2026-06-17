package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
				SourceQueries            map[string]struct {
					Query string `json:"query"`
				} `json:"sourceQueries"`
				ExtractionSchema struct {
					Fields []struct {
						Name string `json:"name"`
					} `json:"fields"`
				} `json:"extractionSchema"`
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

func TestExecuteProtocolPlanSourcesJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "protocol", "plan-sources", "--type", "pico", "--question", "Do catalysts improve hydrogen evolution?", "--population", "hydrogen evolution", "--intervention", "catalysts", "--outcome", "efficiency"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			SourcePlan struct {
				Sources []struct {
					Source                   string `json:"source"`
					ReviewerApprovalRequired bool   `json:"reviewerApprovalRequired"`
					CLICommand               string `json:"cliCommand"`
				} `json:"sources"`
			} `json:"sourcePlan"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json decode: %v\n%s", err, stdout.String())
	}
	if !env.OK || len(env.Data.SourcePlan.Sources) < 13 {
		t.Fatalf("unexpected source plan: %#v", env)
	}
	for _, want := range []string{"nasa-ads", "doaj", "core", "zotero", "jabref", "local"} {
		if !sourcePlanHas(env.Data.SourcePlan.Sources, want) {
			t.Fatalf("missing source %q in %s", want, stdout.String())
		}
	}
}

func TestExecuteProtocolLiveSmokeSnapshotWritesStorage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshots", "latest.json")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"protocol", "live-smoke-snapshot", "--output", path, "--connector", "openalex", "--status", "pass", "--message", "ok", "--fields", "source,query,work_id,raw_ref"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("snapshot was not written: %v", err)
	}
	if !strings.Contains(string(payload), `"connectorId": "openalex"`) || !strings.Contains(string(payload), `"status": "pass"`) {
		t.Fatalf("snapshot payload missing openalex pass result:\n%s", string(payload))
	}
}

func TestExecuteProtocolCapabilitiesJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "protocol", "capabilities"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			Registry struct {
				Connectors []struct {
					ID                        string   `json:"id"`
					SupportedEntities         []string `json:"supportedEntities"`
					RateLimitPolicy           string   `json:"rateLimitPolicy"`
					AuthNeeds                 string   `json:"authNeeds"`
					LiveSmokeStatus           string   `json:"liveSmokeStatus"`
					LicenseShareabilityPolicy string   `json:"licenseShareabilityPolicy"`
					Cacheability              string   `json:"cacheability"`
					ProvenanceFields          []string `json:"provenanceFields"`
				} `json:"connectors"`
			} `json:"registry"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json decode: %v\n%s", err, stdout.String())
	}
	if !env.OK || len(env.Data.Registry.Connectors) < 13 {
		t.Fatalf("unexpected registry: %#v", env)
	}
	for _, connector := range env.Data.Registry.Connectors {
		if connector.ID == "openalex" && (len(connector.SupportedEntities) == 0 || connector.RateLimitPolicy == "" || connector.AuthNeeds == "" || connector.LiveSmokeStatus == "" || connector.LicenseShareabilityPolicy == "" || connector.Cacheability == "" || len(connector.ProvenanceFields) == 0) {
			t.Fatalf("openalex capability incomplete: %#v", connector)
		}
	}
}

func sourcePlanHas(entries []struct {
	Source                   string `json:"source"`
	ReviewerApprovalRequired bool   `json:"reviewerApprovalRequired"`
	CLICommand               string `json:"cliCommand"`
}, source string) bool {
	for _, entry := range entries {
		if entry.Source == source && entry.ReviewerApprovalRequired && entry.CLICommand != "" {
			return true
		}
	}
	return false
}
