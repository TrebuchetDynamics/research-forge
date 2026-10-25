package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/research"
)

func TestCompareDuplicateLabelsCLI(t *testing.T) {
	for _, mode := range []string{"text", "json"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			a, b := filepath.Join(root, "a", "run"), filepath.Join(root, "b", "run")
			writeFixtureSearchDir(t, a, "a", []map[string]any{
				{"title": "First", "identifiers": map[string]any{"DOI": "10.1/first"}},
			})
			writeFixtureSearchDir(t, b, "b", []map[string]any{
				{"title": "Second", "identifiers": map[string]any{"DOI": "10.1/second"}},
			})
			out := filepath.Join(root, "comparison.md")
			if err := os.WriteFile(out, []byte("sentinel"), 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{"compare", a, b, "--out", out}
			if mode == "json" {
				args = append([]string{"--json"}, args...)
			}
			for _, labels := range [][]string{nil, {"--labels", "same,same"}} {
				var stdout, stderr bytes.Buffer
				code := Execute(append(append([]string{}, args...), labels...), &stdout, &stderr)
				if code != 2 {
					t.Fatalf("exit = %d, want 2: stdout=%s stderr=%s", code, &stdout, &stderr)
				}
				message := stderr.String()
				if mode == "json" {
					var envelope struct {
						OK    bool                           `json:"ok"`
						Error struct{ Code, Message string } `json:"error"`
					}
					if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil || envelope.OK || envelope.Error.Code != "duplicate_labels" {
						t.Fatalf("error envelope = %s, %v", &stdout, err)
					}
					message = envelope.Error.Message
				}
				if !strings.Contains(message, "--labels") || !strings.Contains(message, "distinct") {
					t.Errorf("error lacks distinct --labels remedy: %s", message)
				}
				if data, err := os.ReadFile(out); err != nil || string(data) != "sentinel" {
					t.Fatalf("output changed: %q, %v", data, err)
				}
			}

			var stdout, stderr bytes.Buffer
			if code := Execute(append(args, "--labels", "first,second"), &stdout, &stderr); code != 0 {
				t.Fatalf("unique labels exit = %d: stdout=%s stderr=%s", code, &stdout, &stderr)
			}
			data, err := os.ReadFile(out)
			if err != nil || !strings.Contains(string(data), "**In all directories:** 0") || !strings.Contains(string(data), "**Unique to one directory:** 2") {
				t.Fatalf("recovered output = %s, %v", data, err)
			}
			if mode == "json" {
				var envelope struct {
					OK   bool                   `json:"ok"`
					Data research.CompareResult `json:"data"`
				}
				if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
					t.Fatal(err)
				}
				want := []research.ComparisonPaper{
					{Title: "First", ID: "10.1/first", In: []string{"first"}},
					{Title: "Second", ID: "10.1/second", In: []string{"second"}},
				}
				if !envelope.OK || envelope.Data.Total != 2 || !reflect.DeepEqual(envelope.Data.Labels, []string{"first", "second"}) || !reflect.DeepEqual(envelope.Data.Papers, want) {
					t.Fatalf("recovered JSON = %s", &stdout)
				}
			} else if !strings.Contains(stdout.String(), string(data)) {
				t.Fatalf("recovered stdout differs from report: %s", &stdout)
			}
		})
	}
}
