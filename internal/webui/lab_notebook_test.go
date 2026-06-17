package webui

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

func TestLabNotebookTimelineCoversMetaAnalysisWorkflowEventCategories(t *testing.T) {
	project := t.TempDir()
	for _, event := range []provenance.Event{
		{ID: "import", Action: "library.import.completed"},
		{ID: "source", Action: "source.refresh.completed"},
		{ID: "parser", Action: "parser.run.completed"},
		{ID: "review", Actor: "reviewer-a", Action: "screening.decision.accepted"},
		{ID: "extraction", Actor: "reviewer-a", Action: "evidence.item.corrected"},
		{ID: "analysis", Action: "analysis.run.completed"},
		{ID: "report", Action: "report.build.completed"},
	} {
		appendNotebookEvent(t, project, event)
	}
	state, err := BuildLabNotebookTimelineState(project)
	if err != nil {
		t.Fatalf("BuildLabNotebookTimelineState: %v", err)
	}
	for _, category := range []string{"imports", "source refreshes", "parser runs", "reviewer decisions", "extraction edits", "analysis reruns", "report builds"} {
		if !hasNotebookCategory(state, category) {
			t.Fatalf("missing category %q in %#v", category, state.Events)
		}
	}
}

func TestLabNotebookTimelineClassifiesHumanAndAutomatedEvents(t *testing.T) {
	project := t.TempDir()
	appendNotebookEvent(t, project, provenance.Event{ID: "evt-auto", Timestamp: "2026-01-01T00:00:00Z", Actor: "rforge", Action: "analysis.run.completed", Target: "run1"})
	appendNotebookEvent(t, project, provenance.Event{ID: "evt-human", Timestamp: "2026-01-01T00:01:00Z", Actor: "reviewer-a", Action: "evidence.item.accepted", Target: "paper1"})
	state, err := BuildLabNotebookTimelineState(project)
	if err != nil {
		t.Fatalf("BuildLabNotebookTimelineState: %v", err)
	}
	if state.TotalEvents != 2 || state.HumanEvents != 1 || state.AutomatedEvents != 1 || len(state.Events) != 2 {
		t.Fatalf("state = %#v", state)
	}
	body := renderHandler(t, NewLabNotebookHandler(state))
	for _, want := range []string{"Lab notebook timeline", "Human workflow events", "Automated workflow events", "analysis.run.completed", "evidence.item.accepted", "reviewer-a"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q:\n%s", want, body)
		}
	}
}

func TestRouterServesLabNotebookAndSnapshot(t *testing.T) {
	project := t.TempDir()
	appendNotebookEvent(t, project, provenance.Event{ID: "evt-human", Actor: "reviewer-a", Action: "source.plan.approved", Target: "source_plan"})
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: project}))
	defer ts.Close()
	body := httpGetBody(t, ts.URL+"/notebook")
	if !strings.Contains(body, "Lab notebook timeline") || !strings.Contains(body, "source.plan.approved") {
		t.Fatalf("/notebook missing timeline: %s", body)
	}
	body = httpGetBody(t, ts.URL+"/notebook/snapshot.json")
	var state LabNotebookTimelineState
	if err := json.Unmarshal([]byte(body), &state); err != nil {
		t.Fatalf("snapshot json: %v\n%s", err, body)
	}
	if state.HumanEvents != 1 {
		t.Fatalf("snapshot = %#v", state)
	}
}

func hasNotebookCategory(state LabNotebookTimelineState, category string) bool {
	for _, event := range state.Events {
		if event.Category == category {
			return true
		}
	}
	return false
}

func appendNotebookEvent(t *testing.T, project string, event provenance.Event) {
	t.Helper()
	event.SchemaVersion = "1"
	if err := provenance.Append(project, event); err != nil {
		t.Fatalf("append event: %v", err)
	}
}
