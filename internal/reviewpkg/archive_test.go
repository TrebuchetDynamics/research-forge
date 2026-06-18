package reviewpkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageArchiveRestoreMoveAuditReplayWithoutPrivateLocalState(t *testing.T) {
	workspace := t.TempDir()
	project := filepath.Join(workspace, "private-project")
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Portable Review'\n")
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
	write(t, filepath.Join(project, "data", "provenance.jsonl"), `{"action":"fixture"}`)
	write(t, filepath.Join(project, "data", "source-plans", "plan.json"), `{"query":"fixture"}`)
	write(t, filepath.Join(project, "data", "import-receipts", "receipt.json"), `{"imported":1}`)
	write(t, filepath.Join(project, "data", "library.json"), `[{"Title":"Portable source record","Identifiers":{"DOI":"10.0000/portable"},"SourceRefs":[{"Source":"fixture"}]}]`)
	write(t, filepath.Join(project, "data", "evidence.items.json"), `[]`)
	write(t, filepath.Join(project, "analysis", "run.json"), `{"InputRows":[]}`)
	write(t, filepath.Join(project, "reports", "report.md"), `# Report`)
	write(t, filepath.Join(project, "cache", "private.tmp"), `secret cache`)

	pkgDir := filepath.Join(workspace, "review.rforgepkg")
	if _, err := Create(project, pkgDir, Options{CreatedBy: "tester"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	archivePath := filepath.Join(workspace, "review.rforgepkg.tar")
	if err := Archive(pkgDir, archivePath); err != nil {
		t.Fatalf("Archive: %v", err)
	}
	restoredParent := filepath.Join(t.TempDir(), "moved")
	if err := Restore(archivePath, restoredParent); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if err := os.RemoveAll(workspace); err != nil {
		t.Fatalf("remove original workspace: %v", err)
	}

	audit, err := Audit(restoredParent)
	if err != nil {
		t.Fatalf("Audit restored: %v", err)
	}
	if !audit.OK {
		t.Fatalf("restored audit = %#v", audit)
	}
	replay, err := Replay(restoredParent)
	if err != nil || !replay.OK {
		t.Fatalf("restored replay = %#v err=%v", replay, err)
	}
	if _, err := os.Stat(filepath.Join(restoredParent, "project", "cache", "private.tmp")); !os.IsNotExist(err) {
		t.Fatalf("private cache restored err=%v", err)
	}
	manifestData, err := os.ReadFile(filepath.Join(restoredParent, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if strings.Contains(string(manifestData), workspace) || strings.Contains(string(manifestData), "private-project") {
		t.Fatalf("manifest leaked private local state: %s", manifestData)
	}
}
