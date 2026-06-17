package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutePackageAuditAndReplay(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Audit Package"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	writeJSONForCLITest(t, filepath.Join(project, "rforge.lock.json"), map[string]string{"version": "1"})
	out := filepath.Join(t.TempDir(), "review.rforgepkg")
	if code := Execute([]string{"--project", project, "package", "create", "--out", out}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("package create code=%d", code)
	}
	var stdout, stderr bytes.Buffer
	if code := Execute([]string{"--json", "package", "audit", out}, &stdout, &stderr); code != 0 {
		t.Fatalf("audit code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"ok":true`) {
		t.Fatalf("audit stdout=%s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "package", "replay", out}, &stdout, &stderr); code != 0 {
		t.Fatalf("replay code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"ok":true`) {
		t.Fatalf("replay stdout=%s", stdout.String())
	}
}
