package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEuropePMCSearchNormalizesBiomedicalRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/webservices/rest/search" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "LightGBM crypto" {
			t.Fatalf("query = %q", r.URL.Query().Get("query"))
		}
		if r.URL.Query().Get("pageSize") != "2" || r.URL.Query().Get("format") != "json" {
			t.Fatalf("query params = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"resultList":{"result":[{"id":"123456","pmid":"123456","pmcid":"PMC1234567","doi":"10.1000/pmc","title":"Machine learning in biomedical finance","authorString":"Smith J, Doe A","journalTitle":"Biomedical ML","pubYear":"2026","abstractText":"Fixture abstract.","isOpenAccess":"Y","license":"cc by","fullTextUrlList":{"fullTextUrl":[{"url":"https://europepmc.org/articles/PMC1234567"},{"url":"https://example.org/fulltext.pdf"}]},"meshHeadingList":{"meshHeading":[{"descriptorName":"Machine Learning"},{"descriptorName":"Biomedical Research"}]}}]}}`))
	}))
	defer server.Close()
	connector := NewEuropePMCConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "LightGBM crypto", Limit: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if response.RawRef != "europepmc:/webservices/rest/search?format=json&pageSize=2&query=LightGBM+crypto" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "europepmc" || record.Identifiers.PMID != "123456" || record.Identifiers.PMCID != "PMC1234567" || record.Identifiers.DOI != "10.1000/pmc" || record.Metadata["mesh_terms"] != "Machine Learning; Biomedical Research" || record.License != "cc by" || len(record.URLs) != 2 {
		t.Fatalf("record = %#v", record)
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords returned error: %v", err)
	}
	if papers[0].Identifiers.PMID != "123456" || papers[0].Identifiers.PMCID != "PMC1234567" || papers[0].Title != "Machine learning in biomedical finance" {
		t.Fatalf("papers = %#v", papers)
	}
}
