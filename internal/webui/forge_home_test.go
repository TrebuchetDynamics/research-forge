package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

func TestForgeHomeTimelineShowsProjectStateGatesJobsAndNextActions(t *testing.T) {
	project := seedProject(t)
	writeJSON(t, filepath.Join(project, "data", "forge-state.json"), map[string]any{"currentState": "source_plan"})
	writeJSON(t, filepath.Join(project, "data", "jobs.json"), []ForgeBackgroundJob{{ID: "job-1", Status: "queued", Command: "rforge search import"}})
	if err := provenance.Append(project, provenance.Event{SchemaVersion: "1", ID: "evt-1", Timestamp: "2026-01-01T00:00:00Z", Actor: "tester", Action: "source.plan.drafted", Target: "data/source-plan.json", Inputs: map[string]any{}, Outputs: map[string]any{}, Warnings: []string{}}); err != nil {
		t.Fatalf("append provenance: %v", err)
	}
	state, err := BuildForgeHomeState(project)
	if err != nil {
		t.Fatalf("BuildForgeHomeState: %v", err)
	}
	if state.ActiveProject == "" || state.CurrentState != "source_plan" || len(state.ProvenanceEvents) != 1 || len(state.BackgroundJobs) != 1 || len(state.BlockedReviewGates) == 0 || len(state.NextSafeActions) == 0 {
		t.Fatalf("state = %#v", state)
	}
	body := renderHandler(t, NewForgeHomeHandler(state))
	for _, want := range []string{"Forge home", "source_plan", "source.plan.drafted", "Blocked review gates", "Background jobs", "Next safe actions", "CLI equivalent", "rforge protocol"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

func TestBuildForgeHomeStateDoesNotReadSymlinkedForgeState(t *testing.T) {
	project := seedProject(t)
	outsidePath := filepath.Join(t.TempDir(), "outside-forge-state.json")
	if err := os.WriteFile(outsidePath, []byte(`{"currentState":"external_private_state"}`), 0o640); err != nil {
		t.Fatalf("write outside forge state: %v", err)
	}
	statePath := filepath.Join(project, "data", "forge-state.json")
	if err := os.Symlink(outsidePath, statePath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	state, err := BuildForgeHomeState(project)
	if err != nil {
		t.Fatalf("BuildForgeHomeState: %v", err)
	}
	if state.CurrentState == "external_private_state" {
		t.Fatalf("BuildForgeHomeState exposed external forge state: %#v", state)
	}
	info, err := os.Lstat(statePath)
	if err != nil {
		t.Fatalf("lstat forge state: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("forge home read replaced symlink: mode=%v", info.Mode())
	}
}

func TestBuildForgeHomeStateDoesNotReadSymlinkedBackgroundJobs(t *testing.T) {
	project := seedProject(t)
	outsidePath := filepath.Join(t.TempDir(), "outside-jobs.json")
	writeJSON(t, outsidePath, []ForgeBackgroundJob{{ID: "external-private-job", Status: "queued", Command: "private command"}})
	jobsPath := filepath.Join(project, "data", "jobs.json")
	if err := os.Symlink(outsidePath, jobsPath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	state, err := BuildForgeHomeState(project)
	if err != nil {
		t.Fatalf("BuildForgeHomeState: %v", err)
	}
	if len(state.BackgroundJobs) != 0 {
		t.Fatalf("BuildForgeHomeState exposed external background jobs: %#v", state.BackgroundJobs)
	}
	info, err := os.Lstat(jobsPath)
	if err != nil {
		t.Fatalf("lstat background jobs: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("forge home jobs read replaced symlink: mode=%v", info.Mode())
	}
}

func TestBuildForgeHomeStateDoesNotSatisfyClaimGateWithSymlink(t *testing.T) {
	projectPath := seedProject(t)
	writeJSON(t, filepath.Join(projectPath, "data", "forge-state.json"), map[string]any{"currentState": "report_build"})
	claimPath := filepath.Join(projectPath, "data", "claim-panel.json")
	if err := os.Remove(claimPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove seeded claim panel: %v", err)
	}
	externalPath := filepath.Join(t.TempDir(), "external-claim-panel.json")
	writeJSON(t, externalPath, map[string]any{"external": "private"})
	if err := os.Symlink(externalPath, claimPath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	state, err := BuildForgeHomeState(projectPath)
	if err != nil {
		t.Fatalf("BuildForgeHomeState: %v", err)
	}
	blocked := false
	for _, gate := range state.BlockedReviewGates {
		if gate.Gate == "claim approval" {
			blocked = true
			break
		}
	}
	if !blocked {
		t.Fatalf("symlinked claim panel satisfied claim gate: %#v", state.BlockedReviewGates)
	}
	if body := renderHandler(t, NewForgeHomeHandler(state)); !strings.Contains(body, "claim approval") {
		t.Fatalf("Forge home omitted blocked claim gate: %s", body)
	}
	if info, err := os.Lstat(claimPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("claim panel symlink changed: info=%v err=%v", info, err)
	}
}

func TestBuildForgeHomeStateAcceptsSourcePlansDirectory(t *testing.T) {
	projectPath := seedProject(t)
	writeJSON(t, filepath.Join(projectPath, "data", "forge-state.json"), map[string]any{"currentState": "source_plan"})
	if err := os.MkdirAll(filepath.Join(projectPath, "data", "source-plans"), 0o755); err != nil {
		t.Fatalf("mkdir source plans: %v", err)
	}

	state, err := BuildForgeHomeState(projectPath)
	if err != nil {
		t.Fatalf("BuildForgeHomeState: %v", err)
	}
	for _, gate := range state.BlockedReviewGates {
		if gate.Gate == "network/API approval" {
			t.Fatalf("source-plans directory did not satisfy source gate: %#v", state.BlockedReviewGates)
		}
	}
}

func TestRouterServesForgeHome(t *testing.T) {
	project := seedProject(t)
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: project}))
	defer ts.Close()
	body := httpGetBody(t, ts.URL+"/forge")
	if !strings.Contains(body, "Forge home") || !strings.Contains(body, "Next safe actions") {
		t.Fatalf("/forge missing home: %s", body)
	}
}

func httpGetBody(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(data)
}
