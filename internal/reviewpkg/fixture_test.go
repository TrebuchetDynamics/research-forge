package reviewpkg

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateArtificialPhotosynthesisFixturePackageIsOfflineReplayable(t *testing.T) {
	out := filepath.Join(t.TempDir(), "review.rforgepkg")
	pkg, err := CreateArtificialPhotosynthesisFixturePackage(out, Options{})
	if err != nil {
		t.Fatalf("CreateArtificialPhotosynthesisFixturePackage: %v", err)
	}
	if pkg.Manifest.Question != ArtificialPhotosynthesisQuestion || pkg.Manifest.PackageID != "rforgepkg-20260102T030405Z" {
		t.Fatalf("manifest = %#v", pkg.Manifest)
	}
	for _, rel := range []string{
		"manifest.json",
		"checksums.sha256",
		"redaction-report.json",
		"project/data/source-plans/artificial-photosynthesis.json",
		"project/data/import-receipts/fake-sources.json",
		"project/data/source-cache/fake-openalex-artificial-photosynthesis.json",
		"project/data/library.json",
		"project/data/reference-manager/fidelity.json",
		"project/data/reference-manager/interchange-matrix.json",
		"project/data/privacy-licensing-review.json",
		"project/data/legal-acquisition-queue.json",
		"project/data/document-assets.json",
		"project/documents/open-access/ap-fixture.txt",
		"project/data/parser-manifests/fake-parser.json",
		"project/data/screening-audit.jsonl",
		"project/data/evidence.items.json",
		"project/analysis/run1-artifact-manifest.json",
		"project/reports/report.md",
	} {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
	report, err := Audit(out)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if !report.OK {
		t.Fatalf("audit failed: %#v", report)
	}
	replay, err := Replay(out)
	if err != nil || !replay.OK {
		t.Fatalf("replay=%#v err=%v", replay, err)
	}
	checksums, err := os.ReadFile(filepath.Join(out, "checksums.sha256"))
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Manifest.LegalAcquisitionRef == "" || len(pkg.Manifest.DocumentAssetRefs) == 0 {
		t.Fatalf("manifest missing acquisition refs: %#v", pkg.Manifest)
	}
	if len(pkg.Manifest.ReferenceManagerReportRefs) == 0 {
		t.Fatalf("manifest missing reference manager refs: %#v", pkg.Manifest)
	}
	if len(pkg.Manifest.SourceRecordRefs) == 0 || len(pkg.Manifest.ImportReceiptRefs) == 0 {
		t.Fatalf("manifest missing source refs: %#v", pkg.Manifest)
	}
	if !strings.Contains(string(checksums), "project/data/evidence.items.json") || !strings.Contains(string(checksums), "project/reports/report.md") || !strings.Contains(string(checksums), "project/data/library.json") {
		t.Fatalf("checksums missing fixture artifacts:\n%s", checksums)
	}
}

func TestReferenceManagerFixtureFailurePreservesSourceImportArtifacts(t *testing.T) {
	project := t.TempDir()
	sourcePath := filepath.Join(project, "data", "connector-capabilities.json")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("create data directory: %v", err)
	}
	sourceBefore := []byte("prior connector capabilities\n")
	if err := os.WriteFile(sourcePath, sourceBefore, 0o600); err != nil {
		t.Fatalf("write prior source artifact: %v", err)
	}
	if err := os.Mkdir(filepath.Join(project, "data", "library.json"), 0o755); err != nil {
		t.Fatalf("create blocked reference-manager target: %v", err)
	}

	err := WriteArtificialPhotosynthesisReferenceManagerFixture(project)
	if err == nil {
		t.Fatal("reference-manager fixture succeeded with blocked library target")
	}
	sourceAfter, readErr := os.ReadFile(sourcePath)
	if readErr != nil {
		t.Fatalf("read source artifact after failure: %v", readErr)
	}
	if !bytes.Equal(sourceAfter, sourceBefore) {
		t.Errorf("source artifact changed before reference-manager failure:\n got: %s\nwant: %s", sourceAfter, sourceBefore)
	}
	info, statErr := os.Stat(sourcePath)
	if statErr != nil {
		t.Fatalf("stat source artifact: %v", statErr)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("source artifact mode = %o, want 600", got)
	}
}

func TestAcquisitionFixtureFailurePreservesReferenceManagerArtifacts(t *testing.T) {
	project := t.TempDir()
	libraryPath := filepath.Join(project, "data", "library.json")
	if err := os.MkdirAll(filepath.Dir(libraryPath), 0o755); err != nil {
		t.Fatalf("create data directory: %v", err)
	}
	libraryBefore := []byte("prior library\n")
	if err := os.WriteFile(libraryPath, libraryBefore, 0o600); err != nil {
		t.Fatalf("write prior library: %v", err)
	}
	if err := os.Mkdir(filepath.Join(project, "data", "document-assets.json"), 0o755); err != nil {
		t.Fatalf("create blocked acquisition target: %v", err)
	}

	err := WriteArtificialPhotosynthesisAcquisitionFixture(project)
	if err == nil {
		t.Fatal("acquisition fixture succeeded with blocked document-assets target")
	}
	libraryAfter, readErr := os.ReadFile(libraryPath)
	if readErr != nil {
		t.Fatalf("read library after failure: %v", readErr)
	}
	if !bytes.Equal(libraryAfter, libraryBefore) {
		t.Errorf("library changed before acquisition failure:\n got: %s\nwant: %s", libraryAfter, libraryBefore)
	}
	info, statErr := os.Stat(libraryPath)
	if statErr != nil {
		t.Fatalf("stat library: %v", statErr)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("library mode = %o, want 600", got)
	}
	if _, statErr := os.Stat(filepath.Join(project, filepath.FromSlash(artificialPhotosynthesisAcquisitionAssetPath))); !os.IsNotExist(statErr) {
		t.Errorf("failed acquisition left text asset: %v", statErr)
	}
}

func TestCompleteFixtureProjectFailurePreservesBaseProjectArtifacts(t *testing.T) {
	project := t.TempDir()
	manifestPath := filepath.Join(project, "rforge.project.toml")
	manifestBefore := []byte("prior project manifest\n")
	if err := os.WriteFile(manifestPath, manifestBefore, 0o600); err != nil {
		t.Fatalf("write prior project manifest: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(project, "data"), 0o755); err != nil {
		t.Fatalf("create data directory: %v", err)
	}
	if err := os.Mkdir(filepath.Join(project, "data", "document-assets.json"), 0o755); err != nil {
		t.Fatalf("create blocked acquisition target: %v", err)
	}

	err := WriteArtificialPhotosynthesisFixtureProject(project)
	if err == nil {
		t.Fatal("complete fixture succeeded with blocked document-assets target")
	}
	manifestAfter, readErr := os.ReadFile(manifestPath)
	if readErr != nil {
		t.Fatalf("read project manifest after failure: %v", readErr)
	}
	if !bytes.Equal(manifestAfter, manifestBefore) {
		t.Errorf("project manifest changed before acquisition failure:\n got: %s\nwant: %s", manifestAfter, manifestBefore)
	}
	info, statErr := os.Stat(manifestPath)
	if statErr != nil {
		t.Fatalf("stat project manifest: %v", statErr)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("project manifest mode = %o, want 600", got)
	}
	for _, path := range []string{filepath.Join(project, "analysis"), filepath.Join(project, "reports"), filepath.Join(project, "documents")} {
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Errorf("failed complete fixture left new directory %s: %v", path, statErr)
		}
	}
}
