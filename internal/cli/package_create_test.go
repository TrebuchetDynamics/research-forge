package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/reviewpkg"
)

func TestExecutePackageCreateWritesReproducibleReviewPackageFormat(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Package"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	writeJSONForCLITest(t, filepath.Join(project, "rforge.lock.json"), map[string]string{"version": "1"})
	for _, dir := range []string{filepath.Join(project, "data", "source-plans"), filepath.Join(project, "analysis"), filepath.Join(project, "reports")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeJSONForCLITest(t, filepath.Join(project, "data", "source-plans", "plan.json"), map[string]string{"query": "test"})
	writeJSONForCLITest(t, filepath.Join(project, "data", "evidence.items.json"), []map[string]string{{"PaperID": "p1"}})
	writeJSONForCLITest(t, filepath.Join(project, "analysis", "run1-artifact-manifest.json"), map[string]string{"runId": "run1"})
	writeJSONForCLITest(t, filepath.Join(project, "reports", "report.md"), map[string]string{"report": "md"})
	out := filepath.Join(t.TempDir(), "review.rforgepkg")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "package", "create", "--out", out, "--created-by", "tester", "--question", "PICO?"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("package create code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var manifest reviewpkg.Manifest
	if err := readJSONFile(filepath.Join(out, "manifest.json"), &manifest); err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if manifest.SchemaVersion != "1" || manifest.SourcePlanRefs == nil || manifest.AcceptedEvidenceRef == "" || len(manifest.AnalysisArtifactRefs) != 1 || len(manifest.ReportRefs) != 1 {
		t.Fatalf("manifest = %#v", manifest)
	}
	if _, err := os.Stat(filepath.Join(out, "checksums.sha256")); err != nil {
		t.Fatalf("missing checksums: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "redaction-report.json")); err != nil {
		t.Fatalf("missing redaction report: %v", err)
	}
}

func TestExecutePackageCreateRejectsMissingLockfile(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Missing Lock"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	if err := os.Remove(filepath.Join(project, "rforge.lock.json")); err != nil {
		t.Fatalf("remove generated lockfile: %v", err)
	}
	out := filepath.Join(t.TempDir(), "review.rforgepkg")
	var stdout, stderr bytes.Buffer

	code := Execute([]string{"--json", "--project", project, "package", "create", "--out", out}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("package create code=%d stdout=%s stderr=%s, want 1", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"code":"package_create_failed"`) || !strings.Contains(stdout.String(), "rforge.lock.json") {
		t.Fatalf("package create output=%s, want missing lockfile error", stdout.String())
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("failed package create left output behind: %v", err)
	}
}
