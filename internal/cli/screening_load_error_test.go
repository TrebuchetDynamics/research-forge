package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteScreenRejectsMalformedDecisionHistory(t *testing.T) {
	project := filepath.Join(t.TempDir(), "review")
	if code := Execute([]string{"project", "create", project, "--title", "Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code=%d", code)
	}
	if code := Execute([]string{"--project", project, "screen", "configure", "--reason", "off-topic"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen configure exit code=%d", code)
	}
	if err := os.WriteFile(screenEventsPath(project), []byte(`[{"PaperID":`), 0o644); err != nil {
		t.Fatalf("write malformed screening history: %v", err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "--project", project, "screen", "conflicts", "--stage", "title_abstract"}, stdout, stderr)
	if code != 1 {
		t.Fatalf("screen conflicts exit code=%d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"code":"screen_load_failed"`) {
		t.Fatalf("screen conflicts did not report screening load failure: %s", stdout.String())
	}
}

func TestExecuteScreenRejectsInvalidDecisionHistoryEvent(t *testing.T) {
	project := filepath.Join(t.TempDir(), "review")
	if code := Execute([]string{"project", "create", project, "--title", "Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code=%d", code)
	}
	if code := Execute([]string{"--project", project, "screen", "configure", "--reason", "off-topic"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("screen configure exit code=%d", code)
	}
	if err := os.WriteFile(screenEventsPath(project), []byte(`[{"PaperID":"paper-1","Stage":"title_abstract","Decision":"include"}]`), 0o644); err != nil {
		t.Fatalf("write invalid screening history: %v", err)
	}

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "--project", project, "screen", "conflicts", "--stage", "title_abstract"}, stdout, stderr)
	if code != 1 {
		t.Fatalf("screen conflicts exit code=%d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"code":"screen_load_failed"`) || !strings.Contains(stdout.String(), "event 1") {
		t.Fatalf("screen conflicts did not identify invalid screening event: %s", stdout.String())
	}
}
