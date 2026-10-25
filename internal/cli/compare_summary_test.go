package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Catches inflated shared counts reaching stdout/artifacts or changing JSON.
func TestCompareSharedCountsCLI(t *testing.T) {
	root := t.TempDir()
	a, b, c := filepath.Join(root, "a"), filepath.Join(root, "b"), filepath.Join(root, "c")
	ab := map[string]any{"title": "AB", "identifiers": map[string]any{"DOI": "10.1/ab"}}
	abc := map[string]any{"title": "ABC", "identifiers": map[string]any{"DOI": "10.1/abc"}}
	writeFixtureSearchDir(t, a, "a", []map[string]any{
		ab, abc, {"title": "A-only", "identifiers": map[string]any{"ArXivID": "2301.00001"}},
	})
	writeFixtureSearchDir(t, b, "b", []map[string]any{ab, abc})
	writeFixtureSearchDir(t, c, "c", []map[string]any{abc})

	const wantSummary = `## Summary

- **Total unique papers across all dirs:** 3
- **In all directories:** 1
- **Unique to one directory:** 1
- **A:** 3 papers (2 shared, 1 unique)
- **B:** 2 papers (2 shared, 0 unique)
- **C:** 1 papers (1 shared, 0 unique)

## Gaps`
	var textReport string
	for _, mode := range []string{"text", "json"} {
		out := filepath.Join(root, mode+".md")
		args := []string{"compare", a, b, c, "--labels", "A,B,C", "--out", out}
		if mode == "json" {
			args = append([]string{"--json"}, args...)
		}
		var stdout, stderr bytes.Buffer
		if code := Execute(args, &stdout, &stderr); code != 0 {
			t.Fatalf("exit = %d: stdout=%s stderr=%s", code, &stdout, &stderr)
		}
		data, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), wantSummary) {
			t.Errorf("saved summary incorrect:\n%s", data)
		}
		if mode == "text" {
			textReport = string(data)
			if !strings.Contains(stdout.String(), textReport) {
				t.Errorf("stdout differs from saved Markdown:\n%s", &stdout)
			}
		} else {
			if string(data) != textReport {
				t.Errorf("text/JSON saved Markdown differs:\n%s", data)
			}
			var envelope map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			var wantData any
			if err := json.Unmarshal([]byte(`{
					"labels":["A","B","C"], "total":3,
					"papers":[
						{"title":"ABC","id":"10.1/abc","in":["A","B","C"]},
						{"title":"AB","id":"10.1/ab","in":["A","B"]},
						{"title":"A-only","id":"2301.00001","in":["A"]}
					]
				}`), &wantData); err != nil {
				t.Fatal(err)
			}
			if envelope["ok"] != true || !reflect.DeepEqual(envelope["data"], wantData) {
				t.Errorf("JSON fields/membership/order = %s", &stdout)
			}
		}
	}
}
