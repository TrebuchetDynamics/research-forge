package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

func TestForgeHomeTimelineShowsProjectStateGatesJobsAndNextActions(t *testing.T) {
	project := seedProject(t)
	writeJSON(t, filepath.Join(project, "data", "forge-state.json"), map[string]any{"currentState": "source_plan"})
	writeJSON(t, filepath.Join(project, "data", "jobs.json"), []ForgeBackgroundJob{{ID: "job-1", Status: "queued", Command: "rforge search import"}})
	if err := provenance.Append(project, provenance.Event{SchemaVersion: "1", ID: "evt-1", Action: "source.plan.drafted", Target: "data/source-plan.json"}); err != nil {
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
