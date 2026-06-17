package reviewpkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuditAndReplayVerifyChecksumsAndRequiredPackageLinks(t *testing.T) {
	project := t.TempDir()
	write(t, filepath.Join(project, "rforge.project.toml"), "title='Review'\n")
	write(t, filepath.Join(project, "rforge.lock.json"), `{"version":"1"}`)
	write(t, filepath.Join(project, "data", "provenance.jsonl"), `{"action":"test"}`)
	write(t, filepath.Join(project, "data", "evidence.items.json"), `[]`)
	write(t, filepath.Join(project, "analysis", "run.json"), `{"InputRows":[]}`)
	write(t, filepath.Join(project, "reports", "report.md"), `# Report`)
	pkgDir := filepath.Join(t.TempDir(), "review.rforgepkg")
	if _, err := Create(project, pkgDir, Options{}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	report, err := Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	if !report.OK || len(report.Checks) == 0 {
		t.Fatalf("report = %#v", report)
	}
	replay, err := Replay(pkgDir)
	if err != nil || !replay.OK {
		t.Fatalf("Replay report=%#v err=%v", replay, err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "project", "reports", "report.md"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err = Audit(pkgDir)
	if err != nil {
		t.Fatalf("Audit tampered: %v", err)
	}
	if report.OK {
		t.Fatalf("expected checksum failure, got %#v", report)
	}
}
