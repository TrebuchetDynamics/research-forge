package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Neither adapter may announce success or write a report for damaged JSONL.
func TestReportResultsMalformedJSONLCLI(t *testing.T) {
	for _, command := range []string{"brief", "compare"} {
		for _, mode := range []string{"text", "json"} {
			for _, output := range []string{"existing", "absent_parent"} {
				t.Run(command+"/"+mode+"/"+output, func(t *testing.T) {
					root := t.TempDir()
					a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
					writeFixtureSearchDir(t, a, "a", nil)
					writeFixtureSearchDir(t, b, "b", nil)
					badDir, wantCode := a, "brief_results_parse_failed"
					args := []string{command, a}
					if command == "compare" {
						args = append(args, b)
						badDir, wantCode = b, "compare_read_1_failed"
					}
					path := filepath.Join(badDir, "results-deduped.jsonl")
					const rows = "{\"title\":\"Good\"}\n\n \t\n{\"title\":\"private-row\",\n{\"title\":\"Later\"}\n"
					if err := os.WriteFile(path, []byte(rows), 0o644); err != nil {
						t.Fatal(err)
					}
					out := filepath.Join(root, "report.md")
					if output == "existing" {
						if err := os.WriteFile(out, []byte("sentinel"), 0o600); err != nil {
							t.Fatal(err)
						}
					} else {
						out = filepath.Join(root, "new", "report.md")
					}
					args = append(args, "--out", out)
					if mode == "json" {
						args = append([]string{"--json"}, args...)
					}
					var stdout, stderr bytes.Buffer
					if code := Execute(args, &stdout, &stderr); code != 1 {
						t.Fatalf("exit = %d, stdout=%s stderr=%s", code, &stdout, &stderr)
					}
					diagnostic := stderr.String()
					if mode == "json" {
						var response struct {
							OK    bool            `json:"ok"`
							Data  json.RawMessage `json:"data"`
							Error struct {
								Code, Message string
							} `json:"error"`
						}
						if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
							t.Fatal(err)
						}
						if response.OK || response.Error.Code != wantCode || len(response.Data) != 0 || stderr.Len() != 0 {
							t.Fatalf("unexpected JSON error: %s stderr=%s", &stdout, &stderr)
						}
						diagnostic = response.Error.Message
					} else if stdout.Len() != 0 {
						t.Errorf("success output on failure: %s", &stdout)
					}
					if !strings.Contains(diagnostic, path+":4:") || strings.Contains(diagnostic, "private-row") {
						t.Errorf("missing path/physical line or exposed row: %s", diagnostic)
					}
					if strings.Contains(stdout.String()+stderr.String(), "wrote ") {
						t.Error("write advice on failure")
					}
					if output == "existing" {
						if data, err := os.ReadFile(out); err != nil || string(data) != "sentinel" {
							t.Errorf("output changed: %v", err)
						}
					} else if _, err := os.Stat(filepath.Dir(out)); !errors.Is(err, os.ErrNotExist) {
						t.Errorf("output parent created: %v", err)
					}
					if data, err := os.ReadFile(path); err != nil || string(data) != rows {
						t.Errorf("input changed: %v", err)
					}
				})
			}
		}
	}
}
