package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPairedUsageAndCaps(t *testing.T) {
	// Catches conflating nested search pages with host invocations, accepting
	// unequal models/caps, or dropping true overrun from the observed output.
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		valid  bool
	}{
		{"nested-pages", func(r map[string]any) { u := r["usage"].(map[string]any); u["search_calls"] = 3; u["tool_calls"] = 1 }, true},
		{"unmatched-model", func(r map[string]any) { r["model"].(map[string]any)["version"] = "other" }, false},
		{"unmatched-caps", func(r map[string]any) { r["caps"].(map[string]any)["tokens"] = 64000 }, false},
		{"unreported-overrun", func(r map[string]any) { r["usage"].(map[string]any)["input_tokens"] = 32001 }, false},
		{"reported-overrun", func(r map[string]any) {
			r["usage"].(map[string]any)["input_tokens"] = 32001
			r["stop_reason"] = "budget_exhausted"
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			private := filepath.Join(root, "private")
			if data, err := invoke(t, "stub", "--set", "../../internal/benchmarks/testdata/fixture-tiny", "--out", private); err != nil {
				t.Fatalf("%v: %s", err, data)
			}
			e := readObject(t, filepath.Join(private, "experiment.json"))
			tc.mutate(e["runs"].([]any)[0].(map[string]any))
			data, _ := json.Marshal(e)
			if err := os.WriteFile(filepath.Join(private, "experiment.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			output, err := invoke(t, "blind", "--private", private, "--out", filepath.Join(root, "blind"))
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v: %s", tc.valid, err, output)
			}
		})
	}
}
