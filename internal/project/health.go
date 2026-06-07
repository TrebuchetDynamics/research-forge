package project

import (
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/research-forge/internal/storage"
)

// HealthCheck is one actionable Research project health check.
type HealthCheck struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Action  string `json:"action"`
}

// HealthReport summarizes Research project health.
type HealthReport struct {
	Checks []HealthCheck `json:"checks"`
}

// CheckHealth checks Research project invariants without checking the host environment.
func CheckHealth(path string) HealthReport {
	return HealthReport{Checks: []HealthCheck{
		fileHealthCheck("project_manifest", filepath.Join(path, "rforge.project.toml")),
		fileHealthCheck("project_lockfile", filepath.Join(path, "rforge.lock.json")),
		sqliteHealthCheck(filepath.Join(path, "data", "rforge.sqlite")),
	}}
}

func sqliteHealthCheck(path string) HealthCheck {
	store, err := storage.Initialize(path)
	if err != nil {
		return HealthCheck{Name: "sqlite", OK: false, Message: err.Error(), Action: "Create or repair the project data/rforge.sqlite database."}
	}
	defer store.Close()
	if err := store.HealthCheck(); err != nil {
		return HealthCheck{Name: "sqlite", OK: false, Message: err.Error(), Action: "Check database file permissions and rerun rforge doctor."}
	}
	return HealthCheck{Name: "sqlite", OK: true, Message: path, Action: "No action needed."}
}

func fileHealthCheck(name, path string) HealthCheck {
	if _, err := os.Stat(path); err != nil {
		return HealthCheck{Name: name, OK: false, Message: err.Error(), Action: "Create the missing project file or run rforge project create."}
	}
	return HealthCheck{Name: name, OK: true, Message: path, Action: "No action needed."}
}
