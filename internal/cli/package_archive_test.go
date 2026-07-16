package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutePackageArchiveRestoreCompatibility(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Archive Package"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	writeJSONForCLITest(t, filepath.Join(project, "rforge.lock.json"), map[string]string{"version": "1"})
	writeAuditablePackageInputs(t, project)
	pkgDir := filepath.Join(t.TempDir(), "review.rforgepkg")
	if code := Execute([]string{"--project", project, "package", "create", "--out", pkgDir}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("package create code=%d", code)
	}
	archivePath := filepath.Join(t.TempDir(), "review.rforgepkg.tar")
	var stdout, stderr bytes.Buffer
	if code := Execute([]string{"--json", "package", "archive", pkgDir, archivePath}, &stdout, &stderr); code != 0 {
		t.Fatalf("archive code=%d stderr=%s", code, stderr.String())
	}
	restored := filepath.Join(t.TempDir(), "restored")
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "package", "restore", archivePath, restored}, &stdout, &stderr); code != 0 {
		t.Fatalf("restore code=%d stderr=%s", code, stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "package", "audit", restored}, &stdout, &stderr); code != 0 {
		t.Fatalf("audit restored code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"ok":true`) {
		t.Fatalf("audit stdout=%s", stdout.String())
	}
}
