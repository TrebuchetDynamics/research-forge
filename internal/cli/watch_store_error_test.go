package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteWatchAddPreservesMalformedStore(t *testing.T) {
	project := filepath.Join(t.TempDir(), "review")
	if code := Execute([]string{"project", "create", project, "--title", "Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code=%d", code)
	}
	path := filepath.Join(project, "data", "watched-searches.json")
	original := []byte(`[{"Name":`)
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("write malformed watch store: %v", err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "--project", project, "watch", "add", "catalysts", "--source", "openalex", "--query", "photocatalyst"}, stdout, stderr)
	if code != 1 {
		t.Fatalf("watch add exit code=%d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"code":"watch_store_read_failed"`) {
		t.Fatalf("watch add did not report store read failure: %s", stdout.String())
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read watch store after failure: %v", err)
	}
	if !bytes.Equal(after, original) {
		t.Fatalf("watch add replaced malformed store: got %q want %q", after, original)
	}
}
