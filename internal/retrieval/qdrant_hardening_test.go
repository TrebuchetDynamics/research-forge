package retrieval

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestEmbeddingProviderRegistryAndComplianceProfiles(t *testing.T) {
	registry := DefaultEmbeddingProviderRegistry()
	local, ok := registry.Provider("deterministic-hash")
	if !ok || local.ProviderKind != "local" || local.ModelID == "" || local.LicenseNotes == "" || local.Compliance.TextEgress != "none" || local.Dimensions != 8 || local.VectorIndexInvalidation == "" || !local.RetrievalBenchmarkCompatible {
		t.Fatalf("local provider = %#v ok=%t", local, ok)
	}
	httpProvider, ok := registry.Provider("http-embedding")
	if !ok || httpProvider.ProviderKind != "remote-or-local-service" || httpProvider.ModelID == "" || httpProvider.LicenseNotes == "" || !httpProvider.Compliance.RequiresConsent || httpProvider.Compliance.TextEgress == "none" || httpProvider.Compliance.RequiredConfig == "" || httpProvider.Compliance.ModelVersionLock == "" || httpProvider.Compliance.Dimensionality == "" || httpProvider.VectorIndexInvalidation == "" || !httpProvider.RetrievalBenchmarkCompatible {
		t.Fatalf("http provider = %#v ok=%t", httpProvider, ok)
	}
	if err := ValidateEmbeddingProviderCompliance("http-embedding:fixture", false, map[string]string{"RFORGE_EMBEDDING_URL": "http://embed"}); err == nil {
		t.Fatalf("expected http provider to require consent")
	}
	if err := ValidateEmbeddingProviderCompliance("http-embedding:fixture", true, map[string]string{"RFORGE_EMBEDDING_URL": "http://embed", "RFORGE_EMBEDDING_MODEL": "fixture-v1"}); err != nil {
		t.Fatalf("consented http provider blocked: %v", err)
	}
}

func TestQdrantRebuildReportLocksModelDimensionsPrivacyAndInvalidation(t *testing.T) {
	var sawDelete bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/collections/researchforge_passages":
			_, _ = w.Write([]byte(`{"result":true}`))
		case "/collections/researchforge_passages/points/delete":
			sawDelete = true
			_, _ = w.Write([]byte(`{"result":{"status":"completed"}}`))
		case "/collections/researchforge_passages/points":
			body := readQdrantBody(t, r)
			if strings.Contains(body, "private passage text") || strings.Contains(body, `"Text":`) || !strings.Contains(body, "TextChecksum") || !strings.Contains(body, "PayloadRedacted") {
				t.Fatalf("payload privacy not enforced: %s", body)
			}
			_, _ = w.Write([]byte(`{"result":{"status":"completed"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	index, err := NewQdrantIndex(QdrantOptions{BaseURL: server.URL, Embeddings: DeterministicEmbedding{Dimensions: 6}, PayloadPrivacy: PayloadPrivacyRedacted, InvalidateBeforeUpsert: true})
	if err != nil {
		t.Fatalf("NewQdrantIndex: %v", err)
	}
	doc := parsing.ParsedDocument{PaperID: "paper-1", Sections: []parsing.Section{{ID: "s1", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "private passage text"}}}}}
	report, err := index.RebuildWithReport([]parsing.ParsedDocument{doc})
	if err != nil {
		t.Fatalf("RebuildWithReport: %v", err)
	}
	if !sawDelete || report.Indexed != 1 || report.Dimension != 6 || report.EmbeddingProvider != "deterministic-hash" || report.PayloadPrivacy != PayloadPrivacyRedacted || !report.InvalidatedBeforeUpsert {
		t.Fatalf("report=%#v sawDelete=%t", report, sawDelete)
	}
}
