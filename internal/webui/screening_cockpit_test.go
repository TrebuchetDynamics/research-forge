package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestBuildScreeningCockpitStateDoesNotReadSymlinkedEvents(t *testing.T) {
	proj := seedProject(t)
	outsidePath := filepath.Join(t.TempDir(), "outside-screening-events.json")
	writeJSON(t, outsidePath, []screening.DecisionEvent{{
		PaperID: "10.1000/ap-1", Stage: screening.StageTitleAbstract,
		Decision: screening.DecisionUncertain, Reviewer: "external-private-reviewer",
	}})
	eventsPath := filepath.Join(proj, "data", "screening.events.json")
	if err := os.Remove(eventsPath); err != nil {
		t.Fatalf("remove project screening events fixture: %v", err)
	}
	if err := os.Symlink(outsidePath, eventsPath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	state, err := BuildScreeningCockpitState(proj)
	if err != nil {
		t.Fatalf("BuildScreeningCockpitState: %v", err)
	}
	if state.Progress.ScreenedRecords != 0 || len(state.UncertainQueue) != 0 {
		t.Fatalf("BuildScreeningCockpitState exposed external screening events: %#v", state)
	}
	info, err := os.Lstat(eventsPath)
	if err != nil {
		t.Fatalf("lstat screening events: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("screening cockpit read replaced symlink: mode=%v", info.Mode())
	}
}

func TestScreeningCockpitFailsClosedOnMalformedEvents(t *testing.T) {
	proj := seedProject(t)
	eventsPath := filepath.Join(proj, "data", "screening.events.json")
	if err := os.WriteFile(eventsPath, []byte("["), 0o644); err != nil {
		t.Fatalf("write malformed screening events: %v", err)
	}

	if _, err := BuildScreeningCockpitState(proj); err == nil {
		t.Fatal("BuildScreeningCockpitState accepted malformed screening events")
	}
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: proj}))
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/screening")
	if err != nil {
		t.Fatalf("get screening cockpit: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read malformed screening response: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("GET /screening with malformed events status = %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "parse screening events") {
		t.Fatalf("malformed screening response lacks parse context: %s", body)
	}
}

func TestScreeningCockpitFailsClosedOnInvalidDecision(t *testing.T) {
	proj := seedProject(t)
	writeJSON(t, filepath.Join(proj, "data", "screening.events.json"), []screening.DecisionEvent{
		{
			PaperID:  "10.1000/ap-1",
			Stage:    screening.StageTitleAbstract,
			Decision: screening.DecisionExclude,
			Reason:   "not-configured",
			Reviewer: "ada",
		},
	})

	if state, err := BuildScreeningCockpitState(proj); err == nil {
		t.Fatalf("BuildScreeningCockpitState accepted invalid screening decision: %#v", state.Progress)
	}
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: proj}))
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/screening")
	if err != nil {
		t.Fatalf("get screening cockpit: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read invalid screening response: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("GET /screening with invalid decision status = %d: %s", resp.StatusCode, body)
	}
	if !strings.Contains(string(body), "validate screening event 1") {
		t.Fatalf("invalid screening response lacks event context: %s", body)
	}
}

func TestScreeningCockpitDoesNotAdvertiseSymlinkedAuditBundle(t *testing.T) {
	proj := seedProject(t)
	externalPath := filepath.Join(t.TempDir(), "external-screening-audit-bundle.json")
	writeJSON(t, externalPath, screening.ScreeningAuditBundle{SchemaVersion: "1", Stage: screening.StageTitleAbstract})
	auditPath := filepath.Join(proj, "data", "screening-audit-bundle.json")
	if err := os.Remove(auditPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove project audit bundle fixture: %v", err)
	}
	if err := os.Symlink(externalPath, auditPath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}

	state, err := BuildScreeningCockpitState(proj)
	if err != nil {
		t.Fatalf("BuildScreeningCockpitState: %v", err)
	}
	if state.HasAuditBundle {
		t.Fatalf("screening cockpit advertised symlinked audit bundle: %#v", state)
	}
	if body := renderHandler(t, NewScreeningCockpitHandler(state)); strings.Contains(body, `<a href="/data/screening-audit-bundle.json">`) {
		t.Fatalf("screening cockpit rendered symlinked audit bundle: %s", body)
	}
	if info, err := os.Lstat(auditPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("audit bundle symlink changed: info=%v err=%v", info, err)
	}
}

func TestRouterServesScreeningAuditBundleDownload(t *testing.T) {
	proj := seedProject(t)
	writeJSON(t, filepath.Join(proj, "data", "screening-audit-bundle.json"), screening.ScreeningAuditBundle{
		SchemaVersion: "1",
		Stage:         screening.StageTitleAbstract,
		InputHash:     "download-v1",
	})
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: proj}))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/data/screening-audit-bundle.json")
	if err != nil {
		t.Fatalf("get screening audit bundle: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read screening audit bundle: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET screening audit bundle status = %d: %s", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("screening audit bundle content type = %q", got)
	}
	if got := resp.Header.Get("Content-Disposition"); got != `attachment; filename="screening-audit-bundle.json"` {
		t.Fatalf("screening audit bundle disposition = %q", got)
	}
	if !strings.Contains(string(body), "download-v1") {
		t.Fatalf("screening audit bundle body missing project data: %s", body)
	}
}

func TestRouterRejectsSymlinkedScreeningAuditBundleDownload(t *testing.T) {
	proj := seedProject(t)
	externalPath := filepath.Join(t.TempDir(), "external-screening-audit-bundle.json")
	writeJSON(t, externalPath, screening.ScreeningAuditBundle{SchemaVersion: "external-private-download"})
	auditPath := filepath.Join(proj, "data", "screening-audit-bundle.json")
	if err := os.Remove(auditPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("remove project audit bundle fixture: %v", err)
	}
	if err := os.Symlink(externalPath, auditPath); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: proj}))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/data/screening-audit-bundle.json")
	if err != nil {
		t.Fatalf("get symlinked screening audit bundle: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read symlinked screening audit response: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET symlinked screening audit bundle status = %d: %s", resp.StatusCode, body)
	}
	if strings.Contains(string(body), "external-private-download") {
		t.Fatalf("screening audit download disclosed external data: %s", body)
	}
	if info, err := os.Lstat(auditPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("audit bundle symlink changed: info=%v err=%v", info, err)
	}
}

func TestScreeningCockpitRejectsTruncatedAuditBundle(t *testing.T) {
	proj := seedProject(t)
	auditPath := filepath.Join(proj, "data", "screening-audit-bundle.json")
	if err := os.WriteFile(auditPath, []byte(`{"schemaVersion":"1"`), 0o644); err != nil {
		t.Fatalf("write truncated screening audit bundle: %v", err)
	}

	state, err := BuildScreeningCockpitState(proj)
	if err != nil {
		t.Fatalf("BuildScreeningCockpitState: %v", err)
	}
	if state.HasAuditBundle {
		t.Fatalf("screening cockpit advertised truncated audit bundle: %#v", state)
	}
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: proj}))
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/data/screening-audit-bundle.json")
	if err != nil {
		t.Fatalf("get truncated screening audit bundle: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET truncated screening audit bundle status = %d: %s", resp.StatusCode, body)
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
