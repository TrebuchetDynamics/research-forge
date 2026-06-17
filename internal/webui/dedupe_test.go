package webui

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

func TestDedupeReviewHandlerRendersClustersDecisionHistoryAndAuditProvenance(t *testing.T) {
	project := t.TempDir()
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	left, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Catalyst A", Identifiers: library.Identifiers{DOI: "10.1000/same"}, Year: 2020, SourceRefs: []library.SourceRef{{Source: "zotero", Metadata: map[string]string{"collections": "Included", "tags": "catalyst", "citation-key": "smith2026cat"}}}})
	right, _ := library.NewPaperRecord(library.PaperRecordInput{Title: "Unrelated title", Identifiers: library.Identifiers{CrossrefID: "10.1000/same", OpenAlexID: "W123"}, Year: 2024, SourceRefs: []library.SourceRef{{Source: "openalex", Metadata: map[string]string{"concepts": "chemistry"}}}})
	if err := store.ReplaceAll([]library.PaperRecord{left, right}); err != nil {
		t.Fatalf("replace: %v", err)
	}
	decision := library.IdentityDecision{ID: "decision-1", ClusterID: "identity-cluster-1", Action: library.IdentityDecisionMerge, Reason: "reviewed", Reversible: true, Before: []library.PaperRecord{left, right}, After: []library.PaperRecord{left}}
	if err := library.AppendIdentityDecision(filepath.Join(project, "data", "identity-decisions.jsonl"), decision); err != nil {
		t.Fatalf("append decision: %v", err)
	}
	if err := provenance.Append(project, provenance.Event{SchemaVersion: "1", ID: "evt-1", Action: "identity.merge.approved", Target: "identity-cluster-1", Outputs: map[string]any{"reversible": true}}); err != nil {
		t.Fatalf("append provenance: %v", err)
	}
	req := httptest.NewRequest("GET", "/dedupe", nil)
	rec := httptest.NewRecorder()
	newDedupeReviewHandler(func() string { return project }).ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, want := range []string{"Dedupe/cluster review", "revtools-inspired", "visual clustering", "duplicate review", "screening triage", "exportable cluster decisions", "identity-cluster-1", "Catalyst A", "Unrelated title", "Conflicting source fields", "Zotero collection/tag context", "citation-key preservation", "Included", "catalyst", "smith2026cat", "Decision history", "decision-1", "PRISMA/audit provenance", "identity.merge.approved", "rforge --json --project", "identity-decision log", "screening audit bundle"} {
		if !strings.Contains(body, want) {
			t.Fatalf("dedupe screen missing %q:\n%s", want, body)
		}
	}
}

func TestRouterServesDedupeRoute(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{}))
	defer ts.Close()
	body, status, _ := getURL(t, ts.URL+"/dedupe")
	if status != 200 {
		t.Fatalf("GET /dedupe status = %d", status)
	}
	if !strings.Contains(body, "Dedupe/cluster review") {
		t.Fatalf("/dedupe missing review screen: %s", body)
	}
}
