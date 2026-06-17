package webui

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageExportCenterPreviewsReviewPackageContentsBeforeCreation(t *testing.T) {
	project := t.TempDir()
	mustWrite(t, filepath.Join(project, "rforge.project.toml"), "title = \"demo\"\n")
	mustWrite(t, filepath.Join(project, "rforge.lock.json"), "{}\n")
	mustWrite(t, filepath.Join(project, "data", "parser-manifests", "grobid.json"), "{}\n")
	mustWrite(t, filepath.Join(project, "analysis", "manifest.json"), "{}\n")
	mustWrite(t, filepath.Join(project, "reports", "report.md"), "# report\n")
	mustWrite(t, filepath.Join(project, "data", "identity-decisions.jsonl"), "{}\n")
	state := BuildPackageExportCenterState(project)
	if len(state.PackageContents) == 0 || len(state.ParserManifests) == 0 || len(state.AnalysisArtifacts) == 0 || len(state.ReportOutputs) == 0 || len(state.ReviewerDecisionLogs) == 0 {
		t.Fatalf("state missing previews: %#v", state)
	}
	rec := httptest.NewRecorder()
	newPackageExportCenterHandler(func() string { return project }).ServeHTTP(rec, httptest.NewRequest("GET", "/package", nil))
	body := rec.Body.String()
	for _, want := range []string{"reproducibility/export Workbench", "Reproducible review package contents", "redaction results", "checksums", "lockfiles", "external-tool versions", "parser manifests", "analysis artifacts", "report outputs", "reviewer decision logs", "before package creation", "grobid.json", "rforge package create"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q:\n%s", want, body)
		}
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
