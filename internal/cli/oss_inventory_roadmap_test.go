package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteOSSInventoryRoadmapReportsCoverageGaps(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(filepath.Join(dir, "alpha.md"), []byte("# Alpha"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"schemaVersion":"1","entries":[{"id":"alpha","name":"Alpha","area":"retrieval","disposition":"pattern-reference","licensePolicy":"study","note":"alpha.md","risk":"low","nextSlice":"Add alpha retrieval."}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	todo := filepath.Join(dir, "TODO.md")
	if err := os.WriteFile(todo, []byte("- [ ] Something unrelated.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "oss", "inventory-roadmap", manifest, "--todo", todo}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var env struct {
		OK   bool `json:"ok"`
		Data struct {
			CoverageGaps []string `json:"coverageGaps"`
			Markdown     string   `json:"markdown"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("json: %v\n%s", err, stdout.String())
	}
	if !env.OK || len(env.Data.CoverageGaps) == 0 || !strings.Contains(env.Data.Markdown, "## retrieval") {
		t.Fatalf("unexpected report: %#v", env)
	}
}
