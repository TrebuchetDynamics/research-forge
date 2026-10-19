package provenance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A declaration cannot substitute for a ledger receipt or intact output bytes.
func TestComputedLevels(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "result.md")
	if err := os.WriteFile(output, []byte("evidence"), 0600); err != nil {
		t.Fatal(err)
	}
	record := map[string]any{"question": "test", "depth": "standard", "sources": []string{}, "rforge_version": map[string]string{"version": "dev", "commit": "test", "date": "test"}, "errors": []string{}}
	run, err := WriteRun(root, "test.run", record, []string{output})
	if err != nil {
		t.Fatal(err)
	}
	runPath := filepath.Join(root, "provenance", "runs", run.RunID+".json")
	original, err := os.ReadFile(runPath)
	if err != nil {
		t.Fatal(err)
	}
	if result, err := Validate(runPath); err != nil || result.Level != EngineVerified {
		t.Fatalf("engine run: %+v, %v", result, err)
	}
	// A later engine run gets a different immutable receipt, without rewriting this one.
	second, err := WriteRun(root, "test.run", record, []string{output})
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(runPath)
	if second.RunID == run.RunID || string(after) != string(original) {
		t.Fatal("run receipt overwritten")
	}
	legacy := map[string]any{}
	if err := json.Unmarshal(original, &legacy); err != nil {
		t.Fatal(err)
	}
	legacy["schema_version"] = "1"
	legacy["run_id"] = "legacy"
	legacy["outputs"] = []string{"result.md"}
	legacyData, _ := json.Marshal(legacy)
	if err := os.WriteFile(filepath.Join(root, "provenance", "runs", "legacy.json"), legacyData, 0600); err != nil {
		t.Fatal(err)
	}
	ledger, err := os.ReadFile(filepath.Join(root, EventsRelativePath))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, level string
		runs        []string
		entries     []map[string]any
		tamper      bool
		eventKind   string
		invented    bool
		want        Level
		wantErr     bool
	}{
		{name: "hand typed engine", level: "engine_verified", wantErr: true},
		{name: "invented run ID", level: "engine_verified", invented: true, wantErr: true},
		{name: "agent fallback", level: "agent_reported", want: AgentReported},
		{name: "verified index", level: "engine_verified", runs: []string{run.RunID}, want: EngineVerified},
		{name: "minimum referenced legacy run", level: "unverified", runs: []string{run.RunID, "legacy"}, want: Unverified},
		{name: "missing engine event", level: "engine_verified", runs: []string{run.RunID}, eventKind: "missing", wantErr: true},
		{name: "agent event cannot verify engine", level: "engine_verified", runs: []string{run.RunID}, eventKind: "agent", wantErr: true},
		{name: "minimum own entry", level: "agent_reported", runs: []string{run.RunID}, entries: []map[string]any{{"provenance_level": "agent_reported"}}, want: AgentReported},
		{name: "minimum unverified entry", level: "unverified", runs: []string{run.RunID}, entries: []map[string]any{{"provenance_level": "unverified"}}, want: Unverified},
		{name: "tampered output", level: "engine_verified", runs: []string{run.RunID}, tamper: true, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			index := map[string]any{"schema_version": "2", "question": "test", "depth": "standard", "sources": []string{}, "timestamp": "2026-01-01T00:00:00Z", "rforge_version": record["rforge_version"], "errors": []string{}, "outputs": []any{}, "provenance_level": tc.level}
			if tc.runs != nil {
				index["runs"] = tc.runs
			}
			if tc.entries != nil {
				index["entries"] = tc.entries
			}
			if tc.invented {
				index["run_id"] = "invented"
				index["outputs"] = []Output{{Path: "result.md", SHA256: "ee8250fb76e094b34b471f13a73dbbe51d1ae142e9df59d7c0d31ec20f0a0a8e"}}
			}
			if tc.eventKind != "" {
				changed := []byte{}
				if tc.eventKind != "missing" {
					changed = []byte(strings.ReplaceAll(string(ledger), `"actor_kind":"engine"`, `"actor_kind":"agent"`))
				}
				if err := os.WriteFile(filepath.Join(root, EventsRelativePath), changed, 0600); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					if err := os.WriteFile(filepath.Join(root, EventsRelativePath), ledger, 0600); err != nil {
						t.Error(err)
					}
				})
			}
			if tc.tamper {
				if err := os.WriteFile(output, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			data, _ := json.Marshal(index)
			path := filepath.Join(root, "provenance.json")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			result, err := Validate(path)
			if (err != nil) != tc.wantErr || (!tc.wantErr && result.Level != tc.want) {
				t.Fatalf("got %+v, %v; want %s error=%v", result, err, tc.want, tc.wantErr)
			}
		})
	}
}
