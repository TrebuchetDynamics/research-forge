package project

import (
	"path/filepath"
	"testing"
)

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
