package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/screening"
)

func TestScreeningCockpitRendersActiveLearningProgressStoppingAndAuditLinks(t *testing.T) {
	proj := seedProject(t)
	store, err := library.OpenStore(filepath.Join(proj, "data", "library.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	if err := store.Create(library.PaperRecord{Title: "Unscreened catalyst", Identifiers: library.Identifiers{DOI: "10.1000/ap-2"}}); err != nil {
		t.Fatalf("seed paper: %v", err)
	}
	writeJSON(t, filepath.Join(proj, "data", "screening.events.json"), []screening.DecisionEvent{
		{PaperID: "10.1000/ap-1", Stage: screening.StageTitleAbstract, Decision: screening.DecisionUncertain, Reviewer: "ada"},
	})
	writeJSON(t, filepath.Join(proj, "data", "screening-audit-bundle.json"), screening.ScreeningAuditBundle{SchemaVersion: "1", Stage: screening.StageTitleAbstract})
	state, err := BuildScreeningCockpitState(proj)
	if err != nil {
		t.Fatalf("BuildScreeningCockpitState: %v", err)
	}
	if len(state.ActiveLearningQueue) == 0 || state.Progress.ScreenedRecords != 1 || len(state.UncertainQueue) != 1 || !state.HasAuditBundle {
		t.Fatalf("state = %#v", state)
	}
	body := renderHandler(t, NewScreeningCockpitHandler(state))
	for _, want := range []string{"Screening cockpit", "ASReview-inspired", "Active-learning queue", "uncertainty", "exploration", "Reviewer assignment", "Conflict/adjudication panels", "Recall/effort curves", "Progress metrics", "Stopping diagnostics", "Exportable audit bundle", "screening-audit-bundle.json"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

func TestRouterServesScreeningCockpit(t *testing.T) {
	proj := seedProject(t)
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: proj}))
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/screening")
	if err != nil {
		t.Fatalf("get /screening: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "Screening cockpit") || !strings.Contains(body, "Active-learning queue") {
		t.Fatalf("/screening missing cockpit: %s", body)
	}
}
