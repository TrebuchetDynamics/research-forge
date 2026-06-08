package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckHealthIsReadOnlyWhenSQLiteIsMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if err := os.MkdirAll(filepath.Join(dir, "data"), 0o755); err != nil {
		t.Fatalf("make data dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rforge.project.toml"), []byte("title = \"Demo\"\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rforge.lock.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}

	report := CheckHealth(dir)

	if _, err := os.Stat(filepath.Join(dir, "data", "rforge.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("CheckHealth created sqlite database, err=%v", err)
	}
	var sqliteCheck HealthCheck
	for _, check := range report.Checks {
		if check.Name == "sqlite" {
			sqliteCheck = check
		}
	}
	if sqliteCheck.Name == "" {
		t.Fatalf("missing sqlite check: %#v", report.Checks)
	}
	if sqliteCheck.OK {
		t.Fatalf("sqlite check OK = true, want false when database is missing")
	}
}

func TestCheckHealthReportsProjectManifestLockfileAndSQLite(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	_, err := Create(dir, CreateOptions{Title: "Demo Review"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	report := CheckHealth(dir)
	if len(report.Checks) != 3 {
		t.Fatalf("len(checks) = %d, want 3: %#v", len(report.Checks), report.Checks)
	}
	seen := map[string]HealthCheck{}
	for _, check := range report.Checks {
		if check.Action == "" {
			t.Fatalf("check missing action: %#v", check)
		}
		seen[check.Name] = check
	}
	for _, name := range []string{"project_manifest", "project_lockfile", "sqlite"} {
		check, ok := seen[name]
		if !ok {
			t.Fatalf("missing check %q in %#v", name, report.Checks)
		}
		if !check.OK {
			t.Fatalf("check %q failed: %#v", name, check)
		}
	}
}
