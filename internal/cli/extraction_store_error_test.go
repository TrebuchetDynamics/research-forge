package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteExtractionSchemaAddPreservesMalformedStore(t *testing.T) {
	project := filepath.Join(t.TempDir(), "review")
	if code := Execute([]string{"project", "create", project, "--title", "Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code=%d", code)
	}
	path := evidenceSchemasPath(project)
	original := []byte(`[{"Name":`)
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("write malformed schema store: %v", err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "--project", project, "extraction", "schema", "add", "outcomes", "--field", "effect:string"}, stdout, stderr)
	if code != 1 {
		t.Fatalf("schema add exit code=%d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"code":"schema_store_read_failed"`) {
		t.Fatalf("schema add did not report store read failure: %s", stdout.String())
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema store after failure: %v", err)
	}
	if !bytes.Equal(after, original) {
		t.Fatalf("schema add replaced malformed store: got %q want %q", after, original)
	}
}

func TestExecuteExtractAddPreservesMalformedEvidenceStore(t *testing.T) {
	project := filepath.Join(t.TempDir(), "review")
	if code := Execute([]string{"project", "create", project, "--title", "Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code=%d", code)
	}
	path := evidenceItemsPath(project)
	original := []byte(`[{"PaperID":`)
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("write malformed evidence store: %v", err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "--project", project, "extract", "add", "--paper", "paper-1", "--schema", "outcomes", "--value", "effect=1", "--support", "passage:p1", "--status", "accepted"}, stdout, stderr)
	if code != 1 {
		t.Fatalf("extract add exit code=%d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"code":"evidence_store_read_failed"`) {
		t.Fatalf("extract add did not report store read failure: %s", stdout.String())
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read evidence store after failure: %v", err)
	}
	if !bytes.Equal(after, original) {
		t.Fatalf("extract add replaced malformed store: got %q want %q", after, original)
	}
}

func TestExecuteEvidenceAuditRejectsMalformedStore(t *testing.T) {
	project := filepath.Join(t.TempDir(), "review")
	if code := Execute([]string{"project", "create", project, "--title", "Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code=%d", code)
	}
	if err := os.WriteFile(evidenceItemsPath(project), []byte(`[{"PaperID":`), 0o644); err != nil {
		t.Fatalf("write malformed evidence store: %v", err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "--project", project, "evidence", "audit"}, stdout, stderr)
	if code != 1 {
		t.Fatalf("evidence audit exit code=%d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"code":"evidence_read_failed"`) {
		t.Fatalf("evidence audit did not report store read failure: %s", stdout.String())
	}
}
