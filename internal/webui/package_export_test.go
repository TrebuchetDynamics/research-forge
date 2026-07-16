package webui

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageExportCenterEmptyProjectDoesNotReadWorkingDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()
	mustWrite(t, filepath.Join(dir, "rforge.lock.json"), "{}\n")
	mustWrite(t, filepath.Join(dir, "data", "identity-decisions.jsonl"), "{}\n")

	state := BuildPackageExportCenterState("")
	if len(state.PackageContents) != 0 || len(state.Lockfiles) != 0 || len(state.ReviewerDecisionLogs) != 0 {
		t.Fatalf("empty project scanned cwd artifacts: %#v", state)
	}
}

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

func TestPackageExportCenterDoesNotListSymlinkedGlobArtifacts(t *testing.T) {
	projectPath := t.TempDir()
	analysisDir := filepath.Join(projectPath, "analysis")
	if err := os.MkdirAll(analysisDir, 0o755); err != nil {
		t.Fatalf("mkdir project analysis: %v", err)
	}
	externalPath := filepath.Join(t.TempDir(), "external-private.json")
	mustWrite(t, externalPath, `{"private":true}`)
	artifactPath := filepath.Join(analysisDir, "external-private.json")
	if err := os.Symlink(externalPath, artifactPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	state := BuildPackageExportCenterState(projectPath)
	for _, artifact := range state.AnalysisArtifacts {
		if artifact == "analysis/external-private.json" {
			t.Fatalf("package preview listed symlinked artifact: %#v", state.AnalysisArtifacts)
		}
	}
	rec := httptest.NewRecorder()
	newPackageExportCenterHandler(func() string { return projectPath }).ServeHTTP(rec, httptest.NewRequest("GET", "/package", nil))
	if body := rec.Body.String(); strings.Contains(body, "external-private.json") {
		t.Fatalf("package preview rendered symlinked artifact: %s", body)
	}
	if info, err := os.Lstat(artifactPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("analysis artifact symlink changed: info=%v err=%v", info, err)
	}
}

func TestPackageExportCenterDoesNotListSymlinkedNamedContent(t *testing.T) {
	projectPath := t.TempDir()
	externalPath := filepath.Join(t.TempDir(), "external-project.toml")
	mustWrite(t, externalPath, "title = \"private\"\n")
	projectFilePath := filepath.Join(projectPath, "rforge.project.toml")
	if err := os.Symlink(externalPath, projectFilePath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	state := BuildPackageExportCenterState(projectPath)
	for _, content := range state.PackageContents {
		if content == "rforge.project.toml" {
			t.Fatalf("package preview listed symlinked named content: %#v", state.PackageContents)
		}
	}
	rec := httptest.NewRecorder()
	newPackageExportCenterHandler(func() string { return projectPath }).ServeHTTP(rec, httptest.NewRequest("GET", "/package", nil))
	if body := rec.Body.String(); strings.Contains(body, "rforge.project.toml") {
		t.Fatalf("package preview rendered symlinked named content: %s", body)
	}
	if info, err := os.Lstat(projectFilePath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("named content symlink changed: info=%v err=%v", info, err)
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
