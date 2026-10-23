package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Catches loss of shared-service usage handling or writes on rejected CLI input.
func TestRankBlankTopicCLI(t *testing.T) {
	// Even if validation regresses, all source URLs point to local fixtures.
	workflowFixtureServer(t)
	for _, topic := range []string{"", "   ", "\t\n", "\u00a0\u2003\u3000"} {
		for _, jsonOutput := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/json=%t", topic, jsonOutput), func(t *testing.T) {
				dir := t.TempDir()
				args := []string{"--project", dir, "rank", topic, "--sources", "openalex", "--out", filepath.Join(dir, "results", "rankings.json")}
				if jsonOutput {
					args = append([]string{"--json"}, args...)
				}
				var out, stderr bytes.Buffer
				if code := Execute(args, &out, &stderr); code != 2 {
					t.Errorf("expected exit 2, got %d: %s %s", code, &out, &stderr)
				}
				if jsonOutput {
					var result struct {
						OK    bool `json:"ok"`
						Error struct {
							Code string `json:"code"`
						} `json:"error"`
					}
					if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.OK || result.Error.Code != "usage" {
						t.Errorf("expected JSON usage error, got %s, %v", &out, err)
					}
					if stderr.Len() != 0 {
						t.Errorf("unexpected JSON stderr: %s", &stderr)
					}
				} else if out.Len() != 0 || !strings.Contains(stderr.String(), "ranking topic is required") {
					t.Errorf("expected text usage error only, got stdout=%s stderr=%s", &out, &stderr)
				}
				entries, err := os.ReadDir(dir)
				if err != nil || len(entries) != 0 {
					t.Errorf("rejected CLI input left result/provenance artifacts: %v, %v", entries, err)
				}
			})
		}
	}
}
