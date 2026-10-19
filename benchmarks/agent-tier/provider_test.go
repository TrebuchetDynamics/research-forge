package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPrepareRequiresResilientProviderPolicy(t *testing.T) {
	root := t.TempDir()
	private := filepath.Join(root, "private")
	set := "../../internal/benchmarks/testdata/fixture-tiny"
	if data, err := invoke(t, "stub", "--set", set, "--out", private); err != nil {
		t.Fatalf("%v: %s", err, data)
	}
	for _, scenario := range []string{"single_provider", "duplicate_provider", "fail_fast", "retry_429"} {
		t.Run(scenario, func(t *testing.T) {
			p := readObject(t, filepath.Join(private, "protocol.json"))
			policy := p["provider_policy"].(map[string]any)
			switch scenario {
			case "single_provider":
				policy["with_sources"] = []string{"crossref"}
			case "duplicate_provider":
				policy["with_sources"] = []string{"crossref", "crossref"}
			case "fail_fast":
				policy["continue_on_error"] = false
			case "retry_429":
				policy["rate_limit_action"] = "retry"
			}
			data, _ := json.Marshal(p)
			file := filepath.Join(root, scenario+".json")
			if err := os.WriteFile(file, data, 0600); err != nil {
				t.Fatal(err)
			}
			if data, err := invoke(t, "prepare", "--set", set, "--protocol", file, "--out", filepath.Join(root, scenario)); err == nil {
				t.Fatalf("unsafe policy accepted: %s", data)
			}
		})
	}
}

func TestProviderReceiptsEnforceCompleteFailureAccounting(t *testing.T) {
	root := t.TempDir()
	private := filepath.Join(root, "private")
	if data, err := invoke(t, "stub", "--set", "../../internal/benchmarks/testdata/fixture-tiny", "--rate-limit-provider", "crossref", "--out", private); err != nil {
		t.Fatalf("%v: %s", err, data)
	}
	file := filepath.Join(private, "experiment.json")
	original, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"missing_receipts", "partial_receipts", "retry_after_429", "skip_without_failure", "hidden_failure", "malformed_response"} {
		t.Run(scenario, func(t *testing.T) {
			var e map[string]any
			if err := json.Unmarshal(original, &e); err != nil {
				t.Fatal(err)
			}
			run := e["runs"].([]any)[0].(map[string]any)
			events := run["provider_events"].([]any)
			switch scenario {
			case "missing_receipts":
				run["provider_events"] = []any{}
			case "partial_receipts":
				run["provider_events"] = []any{events[1], events[2], events[4], events[5]}
			case "retry_after_429":
				events[1].(map[string]any)["provider"] = "crossref"
			case "skip_without_failure":
				events[0].(map[string]any)["action"] = "skipped"
			case "hidden_failure":
				events[0].(map[string]any)["action"] = "completed"
			case "malformed_response":
				event := events[1].(map[string]any)
				event["action"], event["error"], event["stop_reason"] = "failed", "invalid JSON response", "error_threshold"
			}
			data, _ := json.Marshal(e)
			if err := os.WriteFile(file, data, 0600); err != nil {
				t.Fatal(err)
			}
			output, err := invoke(t, "blind", "--private", private, "--out", filepath.Join(root, scenario))
			_ = os.Remove(filepath.Join(private, "mapping.json"))
			wantValid := scenario == "malformed_response"
			if (err == nil) != wantValid || (!wantValid && !strings.Contains(string(output), "protocol_violation")) {
				t.Fatalf("valid=%v err=%v: %s", wantValid, err, output)
			}
		})
	}
}

func TestProviderOutageSurvivesBlindScoring(t *testing.T) {
	// Catches lost 429 receipts, retrying a quarantined provider, or dropping
	// outage-affected observations/flags when joining frozen blind scores.
	root := t.TempDir()
	private, blind := filepath.Join(root, "private"), filepath.Join(root, "blind")
	scores, out := filepath.Join(root, "scores.json"), filepath.Join(root, "report.json")
	set := "../../internal/benchmarks/testdata/fixture-tiny"
	if data, err := invoke(t, "stub", "--set", set, "--rate-limit-provider", "crossref", "--out", private); err != nil {
		t.Fatalf("%v: %s", err, data)
	}
	// Preserve one pair affected in both conditions, one affected only with
	// tools, and one unaffected pair. This catches asymmetric-outage masking.
	file := filepath.Join(private, "experiment.json")
	experiment := readObject(t, file)
	for _, value := range experiment["runs"].([]any) {
		run := value.(map[string]any)
		slot := run["slot"].(map[string]any)
		if slot["repetition"] == float64(3) || (slot["repetition"] == float64(2) && slot["condition"] == "without_rforge") {
			for _, value := range run["provider_events"].([]any) {
				e := value.(map[string]any)
				e["action"], e["http_status"], e["retry_after"], e["error"], e["stop_reason"] = "completed", 200, "", "", "completed"
			}
			run["usage"].(map[string]any)["search_calls"] = 6
			run["usage"].(map[string]any)["http_requests"] = 6
		}
	}
	data, _ := json.Marshal(experiment)
	if err := os.WriteFile(file, data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"blind", "--private", private, "--out", blind},
		{"score", "--set", set, "--blind", blind, "--out", scores},
		{"report", "--set", set, "--private", private, "--blind", blind, "--scores", scores, "--out", out},
	} {
		if data, err := invoke(t, args...); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, data)
		}
	}
	experiment = readObject(t, file)
	for _, value := range experiment["runs"].([]any) {
		run := value.(map[string]any)
		failures, skips, successes := 0, 0, 0
		for _, value := range run["provider_events"].([]any) {
			e := value.(map[string]any)
			if e["provider"] == "openalex" {
				t.Fatal("default comparison depends on OpenAlex")
			}
			switch e["action"] {
			case "failed":
				failures++
				if e["provider"] != "crossref" || e["http_status"] != float64(429) || e["retry_after"] != "3600" || e["stop_reason"] != "error_threshold" {
					t.Fatalf("lost rate-limit response: %v", e)
				}
			case "skipped":
				skips++
			case "completed":
				successes++
			}
		}
		slot := run["slot"].(map[string]any)
		affected := slot["repetition"] == float64(1) || (slot["repetition"] == float64(2) && slot["condition"] == "with_rforge")
		wantFailures, wantSuccesses := 0, 6
		if affected {
			wantFailures, wantSuccesses = 1, 4
		}
		if failures != wantFailures || skips != wantFailures || successes != wantSuccesses || run["stop_reason"] != "completed" {
			t.Fatalf("provider outage consumed run or retried: %v", run)
		}
	}
	report := readObject(t, out)
	if report["status"] != "complete" || len(report["runs"].([]any)) != 6 {
		t.Fatal("outage observations were dropped")
	}
	availability := report["provider_availability"].([]any)
	if len(availability) != 6 {
		t.Fatalf("missing availability matrix: %v", availability)
	}
	for _, value := range availability {
		row := value.(map[string]any)
		slot := row["slot"].(map[string]any)
		wantLost := slot["repetition"] == float64(1) || (slot["repetition"] == float64(2) && slot["condition"] == "with_rforge")
		if row["provider_lost"] != wantLost || row["pair_provider_lost"] != (slot["repetition"] != float64(3)) {
			t.Fatalf("unflagged outage: %v", row)
		}
		matched := false
		for _, value := range experiment["runs"].([]any) {
			raw := value.(map[string]any)
			if reflect.DeepEqual(raw["slot"], slot) {
				matched = true
				if !reflect.DeepEqual(row["events"], raw["provider_events"]) {
					t.Fatalf("report altered provider receipts for %v", slot)
				}
			}
		}
		if !matched {
			t.Fatalf("availability without a raw run: %v", slot)
		}
	}
	answers := readObject(t, filepath.Join(blind, "answers.json"))
	if _, leaked := answers["provider_availability"]; leaked {
		t.Fatal("availability leaked to scorer")
	}
}
