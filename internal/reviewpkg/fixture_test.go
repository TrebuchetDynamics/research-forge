package reviewpkg

import (
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
