package reviewpkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestCreateSanitizesPackagedForgeStateProjectPath(t *testing.T) {
	project := t.TempDir()
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
	write(t, filepath.Join(project, "data", "forge-state.json"), `{"schemaVersion":"1","projectPath":`+fmt.Sprintf("%q", project)+`,"currentState":"package_export"}`)
	write(t, filepath.Join(project, "data", "provenance.jsonl"), `{"action":"test"}`)
	write(t, filepath.Join(project, "data", "evidence.items.json"), `[]`)
	write(t, filepath.Join(project, "analysis", "run.json"), `{"InputRows":[]}`)
	write(t, filepath.Join(project, "reports", "report.md"), `# Report`)
	pkgDir := filepath.Join(t.TempDir(), "review.rforgepkg")
	if _, err := Create(project, pkgDir, Options{}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(pkgDir, "project", "data", "forge-state.json"))
	if err != nil {
		t.Fatalf("read packaged forge state: %v", err)
	}
	var state struct {
		ProjectPath string `json:"projectPath"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("parse packaged forge state: %v", err)
	}
	if state.ProjectPath != "project" {
		t.Fatalf("packaged projectPath=%q, want project", state.ProjectPath)
	}
	report, err := Audit(pkgDir)
	if err != nil || !report.OK {
		t.Fatalf("Audit=%#v err=%v", report, err)
	}
}

func TestCreateSkipsSymlinkedProjectFiles(t *testing.T) {
	project := t.TempDir()
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
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

func TestCreateRejectsNonRegularNamedProjectArtifact(t *testing.T) {
	project := t.TempDir()
	if err := os.Mkdir(filepath.Join(project, "rforge.project.toml"), 0o755); err != nil {
		t.Fatalf("create non-regular project manifest: %v", err)
	}
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
	out := filepath.Join(t.TempDir(), "review.rforgepkg")

	if _, err := Create(project, out, Options{}); err == nil {
		t.Fatal("Create accepted a non-regular required project manifest")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("failed Create left package output behind: %v", err)
	}
}

func TestCreateRequiresProjectManifestAndLockfile(t *testing.T) {
	for _, tt := range []struct {
		name    string
		present string
		missing string
	}{
		{name: "project manifest", present: "rforge.lock.json", missing: "rforge.project.toml"},
		{name: "lockfile", present: "rforge.project.toml", missing: "rforge.lock.json"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			project := t.TempDir()
			write(t, filepath.Join(project, tt.present), "required input\n")
			out := filepath.Join(t.TempDir(), "review.rforgepkg")

			_, err := Create(project, out, Options{})
			if err == nil || !strings.Contains(err.Error(), tt.missing) {
				t.Fatalf("Create missing %s error=%v", tt.missing, err)
			}
			if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
				t.Fatalf("failed Create left package output behind: %v", statErr)
			}
		})
	}
}

func TestCreateCopiesGlobbedArtifactsFromProjectPathWithMetaCharacters(t *testing.T) {
	project := filepath.Join(t.TempDir(), "project[1]")
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
	write(t, filepath.Join(project, "analysis", "run.json"), `{"InputRows":[]}`)
	write(t, filepath.Join(project, "reports", "report.md"), "# Report\n")
	out := filepath.Join(t.TempDir(), "review.rforgepkg")

	pkg, err := Create(project, out, Options{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(pkg.Manifest.AnalysisArtifactRefs) != 1 || len(pkg.Manifest.ReportRefs) != 1 {
		t.Fatalf("globbed refs omitted for project path with metacharacters: analysis=%v reports=%v", pkg.Manifest.AnalysisArtifactRefs, pkg.Manifest.ReportRefs)
	}
	for _, rel := range []string{"project/analysis/run.json", "project/reports/report.md"} {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("missing copied artifact %s: %v", rel, err)
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

func TestCreateRestoresExistingPackageAfterLateWriteFailure(t *testing.T) {
	project := t.TempDir()
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
	outputParent := t.TempDir()
	out := filepath.Join(outputParent, "review.rforgepkg")
	write(t, filepath.Join(out, "prior-package.txt"), "existing package content\n")

	_, err := Create(project, out, Options{Clock: func() time.Time {
		activeBuildPath := out
		entries, readErr := os.ReadDir(outputParent)
		if readErr != nil {
			t.Fatalf("read output parent during package build: %v", readErr)
		}
		for _, entry := range entries {
			candidate := filepath.Join(outputParent, entry.Name())
			if candidate != out {
				activeBuildPath = candidate
				break
			}
		}
		if removeErr := os.RemoveAll(activeBuildPath); removeErr != nil {
			t.Fatalf("remove active package build: %v", removeErr)
		}
		if writeErr := os.WriteFile(activeBuildPath, []byte("block package writes"), 0o644); writeErr != nil {
			t.Fatalf("block active package build: %v", writeErr)
		}
		return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	}})
	if err == nil {
		t.Fatal("Create returned nil error after the active package build became unwritable")
	}
	data, readErr := os.ReadFile(filepath.Join(out, "prior-package.txt"))
	if readErr != nil {
		t.Fatalf("read restored prior package: %v", readErr)
	}
	if string(data) != "existing package content\n" {
		t.Fatalf("restored prior package = %q", data)
	}
	entries, readErr := os.ReadDir(outputParent)
	if readErr != nil {
		t.Fatalf("read output parent after failure: %v", readErr)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(out) {
		t.Fatalf("output parent entries after failure = %#v, want only %s", entries, filepath.Base(out))
	}
}

func TestCreateReplacesExistingPackageWithoutTransactionDebris(t *testing.T) {
	project := t.TempDir()
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
	outputParent := t.TempDir()
	out := filepath.Join(outputParent, "review.rforgepkg")
	write(t, filepath.Join(out, "prior-package.txt"), "existing package content\n")

	if _, err := Create(project, out, Options{}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "prior-package.txt")); !os.IsNotExist(err) {
		t.Fatalf("replacement retained prior package content: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "manifest.json")); err != nil {
		t.Fatalf("replacement package missing manifest: %v", err)
	}
	entries, err := os.ReadDir(outputParent)
	if err != nil {
		t.Fatalf("read output parent: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(out) {
		t.Fatalf("output parent entries = %#v, want only %s", entries, filepath.Base(out))
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
