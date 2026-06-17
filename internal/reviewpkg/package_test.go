package reviewpkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateReviewPackageFormatIncludesManifestRedactionAndChecksums(t *testing.T) {
	project := t.TempDir()
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
	for _, rel := range []string{
		"data/source-plans/plan.json", "data/identity-decisions.jsonl", "data/parser-manifests/grobid.json", "data/screening-audit.jsonl", "data/evidence.schemas.json", "data/evidence.items.json", "analysis/run1-artifact-manifest.json", "reports/report.md",
	} {
		write(t, filepath.Join(project, rel), rel)
	}
	out := filepath.Join(t.TempDir(), "review.rforgepkg")
	pkg, err := Create(project, out, Options{CreatedBy: "tester", Question: "PICO?"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if pkg.Manifest.SchemaVersion != "1" || pkg.Manifest.ProjectManifestRef == "" || pkg.Manifest.LockfileRef == "" || pkg.Manifest.ChecksumManifestRef != "checksums.sha256" {
		t.Fatalf("manifest = %#v", pkg.Manifest)
	}
	for _, rel := range []string{"manifest.json", "checksums.sha256", "redaction-report.json", "project/rforge.project.toml", "project/data/source-plans/plan.json", "project/analysis/run1-artifact-manifest.json", "project/reports/report.md"} {
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

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
