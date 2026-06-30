package reviewpkg

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type closeErrWriter struct{ closeErr error }

func (w *closeErrWriter) Write(p []byte) (int, error) { return len(p), nil }
func (w *closeErrWriter) Close() error                { return w.closeErr }

func TestCopyAndCloseReturnsCloseError(t *testing.T) {
	w := &closeErrWriter{closeErr: errors.New("simulated flush failure")}
	if err := copyAndClose(w, strings.NewReader("data")); err == nil {
		t.Fatalf("copyAndClose returned nil error despite a failing package file Close")
	}
}

func TestCreateReviewPackageFormatIncludesManifestRedactionAndChecksums(t *testing.T) {
	project := t.TempDir()
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
	for _, rel := range []string{
		"data/retrieval.lock.json", "data/source-plans/plan.json", "data/identity-decisions.jsonl", "data/parser-manifests/grobid.json", "data/screening-audit.jsonl", "data/evidence.schemas.json", "data/evidence.items.json", "analysis/run1-artifact-manifest.json", "reports/report.md",
	} {
		write(t, filepath.Join(project, rel), rel)
	}
	out := filepath.Join(t.TempDir(), "review.rforgepkg")
	pkg, err := Create(project, out, Options{CreatedBy: "tester", Question: "PICO?"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if pkg.Manifest.SchemaVersion != "1" || pkg.Manifest.ProjectManifestRef == "" || pkg.Manifest.LockfileRef == "" || len(pkg.Manifest.LockfileRefs) < 2 || pkg.Manifest.ChecksumManifestRef != "checksums.sha256" || pkg.Manifest.PackageRole != "meta-analysis-spine-first-done-artifact" {
		t.Fatalf("manifest = %#v", pkg.Manifest)
	}
	for _, rel := range []string{"manifest.json", "checksums.sha256", "redaction-report.json", "replay.sh", "audit-report.json", "project/rforge.project.toml", "project/data/retrieval.lock.json", "project/data/source-plans/plan.json", "project/analysis/run1-artifact-manifest.json", "project/reports/report.md"} {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
	checksums, err := os.ReadFile(filepath.Join(out, "checksums.sha256"))
	if err != nil {
		t.Fatalf("read checksums: %v", err)
	}
	if !strings.Contains(string(checksums), "manifest.json") || !strings.Contains(string(checksums), "project/reports/report.md") {
		t.Fatalf("checksums = %s", checksums)
	}
}

func TestCreateSkipsSymlinkedProjectFiles(t *testing.T) {
	project := t.TempDir()
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	secret := filepath.Join(t.TempDir(), "secret.json")
	write(t, secret, `{"token":"secret"}`)
	link := filepath.Join(project, "data", "source-plans", "leak.json")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	out := filepath.Join(t.TempDir(), "review.rforgepkg")
	pkg, err := Create(project, out, Options{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "project", "data", "source-plans", "leak.json")); !os.IsNotExist(err) {
		t.Fatalf("symlink target copied into package: err=%v", err)
	}
	for _, ref := range pkg.Manifest.SourcePlanRefs {
		if strings.Contains(ref, "leak.json") {
			t.Fatalf("symlinked source plan recorded in manifest: %#v", pkg.Manifest.SourcePlanRefs)
		}
	}
}

func TestCreateRejectsDangerousPackageOutputTargets(t *testing.T) {
	project := t.TempDir()
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	for _, out := range []string{project, filepath.Dir(project)} {
		if _, err := Create(project, out, Options{}); err == nil {
			t.Fatalf("Create accepted dangerous output target %s", out)
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Create(project, cwd, Options{}); err == nil {
		t.Fatalf("Create accepted cwd output target %s", cwd)
	}
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
