package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBriefResultsReadFailureCLI(t *testing.T) {
	for _, mode := range []string{"text", "json"} {
		for _, input := range []string{"missing", "directory"} {
			for _, output := range []string{"existing", "absent_parent"} {
				t.Run(mode+"/"+input+"/"+output, func(t *testing.T) {
					root := t.TempDir()
					dir := filepath.Join(root, "search")
					writeFixtureSearchDir(t, dir, "fixture", nil)
					resultsPath := filepath.Join(dir, "results-deduped.jsonl")
					if err := os.Remove(resultsPath); err != nil {
						t.Fatal(err)
					}
					if input == "directory" {
						if err := os.Mkdir(resultsPath, 0o755); err != nil {
							t.Fatal(err)
						}
					}
					out := filepath.Join(root, "brief.md")
					if output == "existing" {
						if err := os.WriteFile(out, []byte("sentinel"), 0o600); err != nil {
							t.Fatal(err)
						}
					} else {
						out = filepath.Join(root, "new", "brief.md")
					}
					args := []string{"brief", dir, "--out", out}
					if mode == "json" {
						args = append([]string{"--json"}, args...)
					}
					var stdout, stderr bytes.Buffer
					if code := Execute(args, &stdout, &stderr); code != 1 {
						t.Fatalf("exit = %d, want 1: stdout=%s stderr=%s", code, &stdout, &stderr)
					}
					message := stderr.String()
					if mode == "json" {
						var envelope struct {
							OK    bool                           `json:"ok"`
							Data  json.RawMessage                `json:"data"`
							Error struct{ Code, Message string } `json:"error"`
						}
						if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil || envelope.OK || envelope.Error.Code != "brief_results_read_failed" || len(envelope.Data) != 0 || stderr.Len() != 0 {
							t.Fatalf("error envelope = %s, stderr=%s, %v", &stdout, &stderr, err)
						}
						message = envelope.Error.Message
					} else if stdout.Len() != 0 {
						t.Errorf("unexpected success output: %s", &stdout)
					}
					if !strings.Contains(message, resultsPath+": ") {
						t.Errorf("error lacks filename/cause: %s", message)
					}
					for _, success := range []string{"# Research Brief", "## Next Steps", "wrote brief", "No deduplicated records found"} {
						if strings.Contains(stdout.String()+stderr.String(), success) {
							t.Errorf("failure includes success report/advice %q", success)
						}
					}
					if output == "existing" {
						if data, err := os.ReadFile(out); err != nil || string(data) != "sentinel" {
							t.Errorf("output changed: %q, %v", data, err)
						}
					} else if _, err := os.Stat(filepath.Dir(out)); !os.IsNotExist(err) {
						t.Errorf("output parent created: %v", err)
					}
				})
			}
		}
	}
}
