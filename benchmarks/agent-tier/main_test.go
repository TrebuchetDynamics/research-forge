package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func invoke(t *testing.T, args ...string) ([]byte, error) {
	t.Helper()
	c := exec.Command("go", append([]string{"run", "."}, args...)...)
	return c.CombinedOutput()
}
func readObject(t *testing.T, p string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err = json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func TestStubProducesBlindScoredReport(t *testing.T) {
	// Catches attribution/counter leakage, dropped duplicate ranks, and a report
	// that fails to join frozen scores back to the paired run matrix.
	root := t.TempDir()
	private, blind, scores, report := filepath.Join(root, "private"), filepath.Join(root, "blind"), filepath.Join(root, "scores.json"), filepath.Join(root, "report.json")
	set := "../../internal/benchmarks/testdata/fixture-tiny"
	for _, args := range [][]string{
		{"stub", "--set", set, "--out", private},
		{"blind", "--private", private, "--out", blind},
		{"score", "--set", set, "--blind", blind, "--out", scores},
		{"report", "--set", set, "--private", private, "--blind", blind, "--scores", scores, "--out", report},
	} {
		if b, err := invoke(t, args...); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, b)
		}
	}
	entries, err := os.ReadDir(blind)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "answers.json" {
		t.Fatalf("scorer mount contains non-answer files: %v", entries)
	}
	raw, _ := os.ReadFile(filepath.Join(blind, "answers.json"))
	for _, marker := range []string{"with_rforge", "without_rforge", "provenance", "run_id", "rforge ", "pi", "input_tokens", "condition"} {
		if strings.Contains(string(raw), marker) {
			t.Fatalf("blind leak: %s", marker)
		}
	}
	r := readObject(t, report)
	if r["status"] != "complete" || r["purpose"] != "fixture" || len(r["runs"].([]any)) != 6 {
		t.Fatalf("incomplete fixture report: %v", r)
	}
	for _, row := range r["runs"].([]any) {
		run := row.(map[string]any)
		want := 1.0 / 3
		if run["condition"] == "with_rforge" {
			want = 2.0 / 3
		}
		if got := run["recall"].(map[string]any)["value"].(float64); got != want {
			t.Fatalf("recall=%v want %v", got, want)
		}
		if run["dedupe_status"] != "unavailable" || run["citation_status"] != "not_implemented" {
			t.Fatal("fabricated unavailable measurements")
		}
	}
	found := false
	for _, row := range r["paired"].([]any) {
		p := row.(map[string]any)
		if p["metric"] == "recall" {
			found = true
			if p["question_count"] != float64(1) || p["repetitions"] != float64(3) || p["ci95_low"] != nil {
				t.Fatalf("wrong sampling unit: %v", p)
			}
		}
	}
	if !found {
		t.Fatal("missing primary contrast")
	}
	// Both extraction boundaries reject tool/condition signatures, including
	// underscore-connected names that a word-boundary regex would miss.
	if err := os.Remove(filepath.Join(private, "mapping.json")); err != nil {
		t.Fatal(err)
	}
	original := readObject(t, filepath.Join(private, "experiment.json"))
	for _, marker := range []string{"condition_with_rforge", "rforge_search", "native_search"} {
		original["runs"].([]any)[0].(map[string]any)["answer"].(map[string]any)["claims"].([]any)[0].(map[string]any)["text"] = marker
		data, _ := json.Marshal(original)
		if err := os.WriteFile(filepath.Join(private, "experiment.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
		if data, err := invoke(t, "blind", "--private", private, "--out", filepath.Join(root, marker)); err == nil {
			t.Fatalf("blind accepted %s: %s", marker, data)
		}
		var injected map[string]any
		_ = json.Unmarshal(raw, &injected)
		injected["answers"].([]any)[0].(map[string]any)["answer"].(map[string]any)["claims"].([]any)[0].(map[string]any)["text"] = marker
		data, _ = json.Marshal(injected)
		_ = os.WriteFile(filepath.Join(blind, "answers.json"), data, 0600)
		if data, err := invoke(t, "score", "--set", set, "--blind", blind, "--out", filepath.Join(root, marker+".json")); err == nil {
			t.Fatalf("score accepted %s: %s", marker, data)
		}
	}
	// The scorer rejects an attribution-bearing extra field, even when it has
	// been placed inside its otherwise condition-free input file.
	var v map[string]any
	_ = json.Unmarshal(raw, &v)
	v["condition"] = "with_rforge"
	bad, _ := json.Marshal(v)
	_ = os.WriteFile(filepath.Join(blind, "answers.json"), bad, 0600)
	if b, err := invoke(t, "score", "--set", set, "--blind", blind, "--out", filepath.Join(root, "bad.json")); err == nil {
		t.Fatalf("scorer accepted condition metadata: %s", b)
	}
}
