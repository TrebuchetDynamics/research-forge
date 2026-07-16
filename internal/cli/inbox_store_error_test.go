package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteInboxTreatsMissingStoreAsEmpty(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Inbox"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "inbox"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("inbox code=%d; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var envelope struct {
		Data struct {
			Items []json.RawMessage `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode inbox output: %v", err)
	}
	if envelope.Data.Items == nil || len(envelope.Data.Items) != 0 {
		t.Fatalf("missing inbox items = %#v, want empty array", envelope.Data.Items)
	}
}

func TestExecuteInboxRejectsMalformedStore(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Inbox"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	if err := os.WriteFile(filepath.Join(project, "data", "inbox.json"), []byte(`[{"id":`), 0o644); err != nil {
		t.Fatalf("write malformed inbox: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "inbox"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("inbox code=%d, want 1; stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"code":"inbox_read_failed"`) {
		t.Fatalf("inbox did not report store read failure: %s", stdout.String())
	}
}
