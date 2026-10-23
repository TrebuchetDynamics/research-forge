package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	b "github.com/TrebuchetDynamics/research-forge/internal/benchmarks"
)

func TestReportDeltaCommand(t *testing.T) {
	fixture, err := b.RunEngine(context.Background(), b.EngineOptions{SetDir: "../../internal/benchmarks/testdata/fixture-tiny", Version: "fixture-version", Revision: "fixture-revision"})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	base, candidate := filepath.Join(dir, "base.json"), filepath.Join(dir, "candidate.json")
	if err = os.WriteFile(base, fixture.Report, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(candidate, fixture.Report, 0600); err != nil {
		t.Fatal(err)
	}
	invoke := func(args ...string) (int, []byte, string) {
		t.Helper()
		var out, stderr bytes.Buffer
		code := run(args, &out, &stderr)
		return code, out.Bytes(), stderr.String()
	}
	for _, path := range []string{base, "../../docs/benchmark/baseline-2026-10-19.json"} {
		t.Run("self "+filepath.Base(path), func(t *testing.T) {
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			args := []string{"--baseline", path, "--candidate", path}
			code, out, stderr := invoke(args...)
			if code != 0 || stderr != "" {
				t.Fatalf("code %d: %s", code, stderr)
			}
			code, again, _ := invoke(args...)
			if code != 0 || !bytes.Equal(out, again) {
				t.Fatal("nondeterministic stdout")
			}
			var decoded b.EngineComparison
			if err = json.Unmarshal(out, &decoded); err != nil {
				t.Fatal(err)
			}
			canonical, err := b.CanonicalJSON(decoded)
			if err != nil || !bytes.Equal(canonical, out) {
				t.Fatalf("not canonical JSON plus LF: %v", err)
			}
			if !reflect.DeepEqual(decoded.Baseline, decoded.Candidate) || decoded.Baseline.Revision == "" || decoded.Baseline.SetChecksum == "" || decoded.Baseline.SnapshotsSHA256 == nil || len(decoded.Limitations) < 3 || len(decoded.Aggregates) == 0 {
				t.Fatalf("missing identity/qualifications: %+v", decoded)
			}
			measured := 0
			for _, a := range decoded.Aggregates {
				if a.Macro.Baseline != nil {
					measured++
					if a.Macro.Delta == nil || *a.Macro.Delta != 0 {
						t.Fatal("nonzero self delta")
					}
				} else if a.Macro.Delta != nil {
					t.Fatal("unavailable became measured")
				}
			}
			if measured == 0 {
				t.Fatal("no measured self deltas")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("input modified")
			}
		})
	}
	mutations := []struct {
		name   string
		change func(map[string]any)
		raw    string
	}{
		{name: "malformed", raw: "{"},
		{name: "duplicate root", raw: strings.Replace(string(fixture.Report), `"mode":"replay"`, `"mode":"replay","mode":"replay"`, 1)},
		{name: "duplicate nested", raw: strings.Replace(string(fixture.Report), `"question_id":"q-one"`, `"question_id":"q-one","question_id":"q-one"`, 1)},
		{name: "trailing JSON", raw: string(fixture.Report) + "{}"},
		{name: "missing root", change: func(m map[string]any) { delete(m, "tier") }},
		{name: "missing nullable", change: func(m map[string]any) { delete(m["runs"].([]any)[0].(map[string]any), "input_tokens") }},
		{name: "missing nested", change: func(m map[string]any) {
			delete(m["runs"].([]any)[0].(map[string]any)["recall"].(map[string]any), "value")
		}},
		{name: "unknown", change: func(m map[string]any) { m["extra"] = true }},
		{name: "null rows", change: func(m map[string]any) { m["runs"] = nil }},
		{name: "inconsistent", change: func(m map[string]any) { m["aggregates"].([]any)[0].(map[string]any)["question_count"] = 10 }},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte(tc.raw)
			if tc.change != nil {
				var m map[string]any
				if err = json.Unmarshal(fixture.Report, &m); err != nil {
					t.Fatal(err)
				}
				tc.change(m)
				data, err = json.Marshal(m)
				if err != nil {
					t.Fatal(err)
				}
			}
			if err = os.WriteFile(candidate, data, 0600); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{{"--baseline", base, "--candidate", candidate}, {"--baseline", candidate, "--candidate", base}} {
				code, out, stderr := invoke(args...)
				if code == 0 || len(out) != 0 || stderr == "" {
					t.Fatalf("rejection emitted success: %d %q %q", code, out, stderr)
				}
			}
			after, err := os.ReadFile(candidate)
			if err != nil || !bytes.Equal(data, after) {
				t.Fatal("rejected input modified")
			}
		})
	}
	for _, args := range [][]string{nil, {"--baseline", base}, {"--baseline", base, "--candidate", base, "positional"}, {"--unknown"}, {"--baseline", base, "--candidate", filepath.Join(dir, "missing")}} {
		code, out, stderr := invoke(args...)
		if code == 0 || len(out) != 0 || stderr == "" {
			t.Fatalf("accepted invalid command: %v %d %q %q", args, code, out, stderr)
		}
	}
	unchanged, err := os.ReadFile(base)
	if err != nil || !bytes.Equal(unchanged, fixture.Report) {
		t.Fatal("baseline modified")
	}
}
