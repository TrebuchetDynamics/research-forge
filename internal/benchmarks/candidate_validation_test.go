package benchmarks_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/benchmarks"
)

// A candidate must validate without inventing approval; opting out of AP may
// bypass only that domain, never counts, other domains, integrity or lifecycle.
func TestValidateCandidateSet(t *testing.T) {
	for _, tc := range []struct {
		name     string
		fallback bool
		edit     func(map[string]any)
		want     string
	}{
		{"strict AP candidate", false, nil, ""},
		{"missing AP requires opt in", false, removeCandidateAP, "invalid_set"},
		{"authorized AP fallback", true, removeCandidateAP, ""},
		{"fallback preserves held out count", true, func(m map[string]any) {
			removeCandidateAP(m)
			m["questions"].([]any)[0].(map[string]any)["split"] = "dev"
		}, "invalid_set"},
		{"fallback preserves dev count", true, func(m map[string]any) {
			m["questions"] = m["questions"].([]any)[:11]
		}, "invalid_set"},
		{"fallback preserves domain count", true, func(m map[string]any) {
			for _, q := range m["questions"].([]any) {
				q.(map[string]any)["domain"] = "medicine"
			}
		}, "invalid_set"},
		{"fixture name cannot bypass candidate composition", true, func(m map[string]any) {
			m["set_id"] = "fixture-candidate"
			m["questions"] = m["questions"].([]any)[:1]
		}, "invalid_set"},
		{"candidate cannot have approval", true, func(m map[string]any) {
			m["approval"] = map[string]any{"approver": "Synthetic", "approved_at": "2020-08-01T00:00:00Z"}
		}, "invalid_set"},
		{"candidate cannot have snapshots", true, func(m map[string]any) {
			m["snapshots_sha256"] = fmt.Sprintf("%064d", 0)
		}, "invalid_set"},
		{"approved input is not a candidate", true, func(m map[string]any) {
			m["state"] = "approved"
		}, "invalid_set"},
		{"fallback still rejects uncapped plans", true, func(m map[string]any) {
			q := m["questions"].([]any)[0].(map[string]any)
			q["search_plan"].([]any)[0].(map[string]any)["filters"] = map[string]any{}
		}, "date_cap_unenforceable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeCandidatePilot(t, tc.edit)
			before, err := os.ReadFile(filepath.Join(dir, "set.json"))
			if err != nil {
				t.Fatal(err)
			}
			err = benchmarks.ValidateCandidateSet(dir, tc.fallback)
			if (err == nil) != (tc.want == "") || (err != nil && err.Error() != tc.want) {
				t.Fatalf("validation = %v; want %q", err, tc.want)
			}
			after, err := os.ReadFile(filepath.Join(dir, "set.json"))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatal("candidate validation rewrote input", err)
			}
			if tc.want == "" {
				_, err := benchmarks.RunEngine(context.Background(), benchmarks.EngineOptions{SetDir: dir})
				if err == nil || err.Error() != "invalid_set" {
					t.Fatalf("candidate became runnable: %v", err)
				}
			}
		})
	}
}

func removeCandidateAP(m map[string]any) {
	m["questions"].([]any)[0].(map[string]any)["domain"] = "medicine"
}

func writeCandidatePilot(t *testing.T, edit func(map[string]any)) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "fixture-tiny", "set.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	template, err := json.Marshal(m["questions"].([]any)[0])
	if err != nil {
		t.Fatal(err)
	}
	questions := []any{}
	for i := range 12 {
		var q map[string]any
		if err := json.Unmarshal(template, &q); err != nil {
			t.Fatal(err)
		}
		q["question_id"] = fmt.Sprintf("q-%c", 'a'+i)
		q["domain"] = []string{"medicine", "education", "energy"}[i%3]
		if i == 0 {
			q["domain"] = "artificial-photosynthesis"
		}
		if i >= 8 {
			q["split"] = "dev"
		}
		r := q["gold_source_review"].(map[string]any)
		r["title"] = fmt.Sprintf("Synthetic review %c", 'a'+i)
		r["identifiers"] = []any{map[string]any{"scheme": "doi", "value": fmt.Sprintf("10.9999/review-%d", i)}}
		for j, g := range q["gold"].([]any) {
			g.(map[string]any)["identifiers"] = []any{map[string]any{"scheme": "doi", "value": fmt.Sprintf("10.9999/paper-%d-%d", i, j)}}
		}
		questions = append(questions, q)
	}
	m["questions"], m["set_id"], m["state"] = questions, "candidate-pilot", "candidate"
	m["approval"], m["snapshots_sha256"] = nil, nil
	sheet := []byte("Synthetic candidate validation sheet\n")
	h := sha256.Sum256(sheet)
	m["review_sheet_sha256"] = hex.EncodeToString(h[:])
	if edit != nil {
		edit(m)
	}
	delete(m, "set_checksum")
	b, err = benchmarks.CanonicalJSON(m)
	if err != nil {
		t.Fatal(err)
	}
	h = sha256.Sum256(bytes.TrimSuffix(b, []byte("\n")))
	m["set_checksum"] = hex.EncodeToString(h[:])
	b, err = benchmarks.CanonicalJSON(m)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "set.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "review.md"), sheet, 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}
