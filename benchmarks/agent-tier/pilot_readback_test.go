package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const publishedPilot = "pilot-pi-2026-10-21"
const pilotSet = "../sets/pool-a"

func TestPublishedPilotReadback(t *testing.T) {
	// Catches scorer drift and inconsistent public rows, summaries and receipts.
	unchangedPilotInputs(t)
	if err := validatePublishedPilot(publishedPilot, t.TempDir()); err != nil {
		t.Fatal(err)
	}
}

func TestPublishedPilotReadbackRejectsTampering(t *testing.T) {
	// Rejection reasons ensure digest checks cannot mask missing semantic guards.
	// Every mutation is confined to a fresh temporary public-package copy.
	unchangedPilotInputs(t)
	first := func(v map[string]any, key string) map[string]any {
		return v[key].([]any)[0].(map[string]any)
	}
	cases := []struct {
		name, file, rejection string
		mutate                func(map[string]any)
	}{
		{"missing report slot", "report.json", "report matrix", func(v map[string]any) { v["runs"] = v["runs"].([]any)[1:] }},
		{"duplicate report slot", "report.json", "report matrix", func(v map[string]any) { rows := v["runs"].([]any); rows[1] = rows[0] }},
		{"missing availability slot", "report.json", "availability matrix", func(v map[string]any) { v["provider_availability"] = v["provider_availability"].([]any)[1:] }},
		{"duplicate availability slot", "report.json", "availability matrix", func(v map[string]any) { rows := v["provider_availability"].([]any); rows[1] = rows[0] }},
		{"protocol digest", "report.json.audit.json", "digest links", func(v map[string]any) { v["protocol_sha256"] = strings.Repeat("0", 64) }},
		{"answer digest", "report.json.audit.json", "digest links", func(v map[string]any) { v["answers_sha256"] = strings.Repeat("0", 64) }},
		{"score digest", "report.json.audit.json", "digest links", func(v map[string]any) { v["scores_sha256"] = strings.Repeat("0", 64) }},
		{"report digest", "report.json.audit.json", "digest links", func(v map[string]any) { v["report_sha256"] = strings.Repeat("0", 64) }},
		{"schedule digest", "registration.json", "digest links", func(v map[string]any) { v["schedule_sha256"] = strings.Repeat("0", 64) }},
		{"incomplete protocol", "protocol.json", "protocol_violation", func(v map[string]any) { v["question_ids"] = v["question_ids"].([]any)[1:] }},
		{"contrast", "report.json", "contrasts", func(v map[string]any) { first(v, "paired")["mean_difference"] = .5 }},
		{"aggregate count is exact", "report.json", "aggregates", func(v map[string]any) { first(v, "aggregates")["question_count"] = 1 }},
		{"aggregate unavailable null", "report.json", "aggregates", func(v map[string]any) { first(v, "aggregates")["macro_mean"] = 0 }},
		{"provider loss flag", "report.json", "availability flags", func(v map[string]any) { first(v, "provider_availability")["provider_lost"] = false }},
		{"paired loss flag", "report.json", "availability flags", func(v map[string]any) { first(v, "provider_availability")["pair_provider_lost"] = false }},
		{"missing unavailable field", "report.json", "report shape", func(v map[string]any) { delete(first(v, "runs"), "dedupe_false_merges") }},
		{"citation null", "report.json", "unavailable facets", func(v map[string]any) { first(v, "runs")["citation_correctness"].(map[string]any)["numerator"] = 0 }},
		{"dedupe null", "report.json", "unavailable facets", func(v map[string]any) { first(v, "runs")["dedupe_true_merges"] = 0 }},
		{"hidden budget stop", "report.json", "stop classification", func(v map[string]any) { first(v, "runs")["stop_reason"] = "completed" }},
		{"native outcome", "report.json", "native classification", func(v map[string]any) { v["runs"].([]any)[3].(map[string]any)["returned"] = 1 }},
		{"native HTTP status", "report.json", "native classification", func(v map[string]any) {
			for _, item := range v["provider_availability"].([]any) {
				row := item.(map[string]any)
				if row["slot"].(map[string]any)["condition"] == "without_rforge" {
					first(row, "events")["http_status"] = 200
					return
				}
			}
		}},
		{"attempt accounting", "report.json", "provider counters", func(v map[string]any) { first(v, "runs")["http_requests"] = 4 }},
		{"frozen score", "scores.json", "frozen bytes", func(v map[string]any) {
			first(v, "scores")["metrics"].(map[string]any)["recall"].(map[string]any)["numerator"] = 1
		}},
		{"question recall analysis", "analysis.json", "analysis", func(v map[string]any) { first(v, "question_recall_differences")["difference"] = .5 }},
		{"sizing unrounded", "analysis.json", "analysis", func(v map[string]any) { v["normal_approximation_question_count_unrounded"] = 240.2 }},
		{"sizing count is exact", "analysis.json", "analysis", func(v map[string]any) { v["planned_observations_at_N3"] = 1446.0000000001 }},
		{"registered sizing parameter", "registration.json", "analysis", func(v map[string]any) { v["analysis_plan"].(map[string]any)["desired_recall_delta"] = .1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := copyPilot(t)
			value := readObject(t, filepath.Join(dir, tc.file))
			tc.mutate(value)
			replacePilotJSON(t, dir, tc.file, value)
			err := validatePublishedPilot(dir, t.TempDir())
			if err == nil || !strings.Contains(err.Error(), tc.rejection) {
				t.Fatalf("got %v; want rejection containing %q", err, tc.rejection)
			}
		})
	}
}

func copyPilot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	entries, err := os.ReadDir(publishedPilot)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(publishedPilot, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, entry.Name()), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func replacePilotJSON(t *testing.T, dir, name string, value any) {
	t.Helper()
	data, err := pilotJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
		t.Fatal(err)
	}
}
