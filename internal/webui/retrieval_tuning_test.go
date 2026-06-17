package webui

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRetrievalTuningHandlerComparesBackendsWithPrivacyAndProvenance(t *testing.T) {
	rec := httptest.NewRecorder()
	newRetrievalTuningHandler(func() string { return t.TempDir() }).ServeHTTP(rec, httptest.NewRequest("GET", "/retrieve", nil))
	body := rec.Body.String()
	for _, want := range []string{"Retrieval tuning", "SQLite FTS", "OpenSearch", "Qdrant vector", "hybrid", "same query", "passage provenance", "ranking explanations", "embedding privacy status", "benchmark scores", "rforge retrieve benchmark"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q:\n%s", want, body)
		}
	}
}
