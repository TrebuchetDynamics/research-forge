package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSearchBatchReceiptFailureRestoresProjectLibrary(t *testing.T) {
	workflowFixtureServer(t)
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new library", true: "existing library"}[existing], func(t *testing.T) {
			project, out := t.TempDir(), t.TempDir()
			path := filepath.Join(project, "data", "library.json")
			if existing {
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("[]\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			blockProvenance(t, out)
			var stdout, stderr bytes.Buffer
			code := Execute([]string{"--project", project, "search", "batch", "--query", "topic", "--sources", "openalex", "--out", out}, &stdout, &stderr)
			if code == 0 {
				t.Fatal("blocked receipt succeeded")
			}
			data, err := os.ReadFile(path)
			if existing && (err != nil || string(data) != "[]\n") || !existing && !os.IsNotExist(err) {
				t.Fatalf("project library survived failed run: %s (%v)", data, err)
			}
		})
	}
}
