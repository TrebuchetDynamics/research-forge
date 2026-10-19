package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInvalidExperimentCannotYieldUsableMetrics(t *testing.T) {
	root := t.TempDir()
	private := filepath.Join(root, "private")
	set := "../../internal/benchmarks/testdata/fixture-tiny"
	if data, err := invoke(t, "stub", "--set", set, "--out", private); err != nil {
		t.Fatalf("%v: %s", err, data)
	}
	e := readObject(t, filepath.Join(private, "experiment.json"))
	e["runs"].([]any)[0].(map[string]any)["gate"].(map[string]any)["reference_review_leak"] = true
	data, _ := json.Marshal(e)
	if err := os.WriteFile(filepath.Join(private, "experiment.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "invalid.json")
	if output, err := invoke(t, "report", "--set", set, "--private", private, "--out", out); err == nil {
		t.Fatalf("leaked experiment succeeded: %s", output)
	}
	report := readObject(t, out)
	if report["status"] != "invalid" || report["aggregates"] != nil || len(report["paired"].([]any)) != 0 || len(report["runs"].([]any)) != 6 {
		t.Fatalf("usable invalid report: %v", report)
	}
	for _, r := range report["runs"].([]any) {
		row := r.(map[string]any)
		if row["recall"].(map[string]any)["value"] != nil || row["input_tokens"] != nil || row["dedupe_status"] != "invalid" {
			t.Fatalf("retained failed metrics: %v", row)
		}
	}
	if report["errors"].([]any)[0].(map[string]any)["code"] != "reference_review_leak" {
		t.Fatal("lost failure code")
	}
	// No validated lock means no invented report/checksum, just a structured
	// diagnostic on stderr with nonzero exit (go run adds its exit-status line).
	output, err := invoke(t, "report", "--set", filepath.Join(root, "missing"), "--private", private, "--out", filepath.Join(root, "missing.json"))
	if err == nil || !strings.Contains(string(output), `{"error":"invalid_set"}`) {
		t.Fatalf("missing JSON diagnostic: %v %s", err, output)
	}
}
