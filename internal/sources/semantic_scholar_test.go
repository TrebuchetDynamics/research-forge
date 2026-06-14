package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSemanticScholarSearchNormalizesPaperRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graph/v1/paper/search" {
			t.Fatalf("path = %s, want /graph/v1/paper/search", r.URL.Path)
		}
		if got := r.URL.Query().Get("query"); got != "limit order book prediction" {
			t.Fatalf("query = %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "2" {
			t.Fatalf("limit = %q", got)
		}
		if got := r.URL.Query().Get("fields"); got == "" {
			t.Fatalf("fields parameter missing")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "total": 12,
  "offset": 0,
  "next": 2,
  "data": [
    {
      "paperId": "649def34f8be52c8b66281af98ae884c09aef38b",
      "title": "DeepLOB: Deep Convolutional Neural Networks for Limit Order Books",
      "abstract": "Limit order book prediction benchmark.",
      "year": 2019,
      "venue": "IEEE Transactions on Signal Processing",
      "url": "https://www.semanticscholar.org/paper/example",
      "isOpenAccess": true,
      "openAccessPdf": {"url": "https://arxiv.org/pdf/1808.03668"},
      "externalIds": {"DOI": "10.1109/TSP.2019.2907260", "ArXiv": "1808.03668"}
    }
  ]
}`))
	}))
	defer server.Close()

	connector := NewSemanticScholarConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "limit order book prediction", Limit: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if response.NextPageCursor != "2" {
		t.Fatalf("NextPageCursor = %q, want 2", response.NextPageCursor)
	}
	if response.RawRef != "semantic-scholar:/graph/v1/paper/search?limit=2&query=limit+order+book+prediction" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "semantic-scholar" || record.SourceID != "649def34f8be52c8b66281af98ae884c09aef38b" {
		t.Fatalf("source identity = %s/%s", record.Source, record.SourceID)
	}
	if record.Identifiers.SemanticScholarID != "649def34f8be52c8b66281af98ae884c09aef38b" {
		t.Fatalf("SemanticScholarID = %q", record.Identifiers.SemanticScholarID)
	}
	if record.Identifiers.DOI != "10.1109/tsp.2019.2907260" || record.Identifiers.ArXivID != "1808.03668" {
		t.Fatalf("identifiers = %+v", record.Identifiers)
	}
	if !record.OpenAccess || len(record.URLs) != 2 {
		t.Fatalf("open access/url normalization failed: open=%t urls=%v", record.OpenAccess, record.URLs)
	}
}

func TestSemanticScholarPaperRecordsPreserveSourceReference(t *testing.T) {
	papers, err := PaperRecords(SourceResponse{RawRef: "semantic-scholar:/graph/v1/paper/search?query=crypto", Records: []SourceRecord{{
		Source:   "semantic-scholar",
		SourceID: "s2-paper",
		Title:    "Forecasting and trading cryptocurrencies with machine learning under changing market conditions",
		Identifiers: Identifiers{
			DOI:               "10.1186/s40854-020-00217-x",
			SemanticScholarID: "s2-paper",
		},
		Year: 2021,
	}}})
	if err != nil {
		t.Fatalf("PaperRecords returned error: %v", err)
	}
	if len(papers) != 1 || papers[0].Identifiers.SemanticScholarID != "s2-paper" {
		t.Fatalf("papers = %#v", papers)
	}
	if len(papers[0].SourceRefs) != 1 || papers[0].SourceRefs[0].Source != "semantic-scholar" {
		t.Fatalf("source refs = %#v", papers[0].SourceRefs)
	}
}
