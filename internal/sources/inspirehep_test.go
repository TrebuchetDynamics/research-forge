package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInspireHEPSearchNormalizesHEPRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/literature" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "Higgs boson" {
			t.Fatalf("q = %q", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("size") != "3" {
			t.Fatalf("size = %q", r.URL.Query().Get("size"))
		}
		_, _ = w.Write([]byte(`{"hits":{"total":1,"hits":[{"id":"1234567","metadata":{"titles":[{"title":"Observation of the Higgs boson"}],"abstracts":[{"value":"We report the observation."}],"dois":[{"value":"10.1016/j.physletb.2012.08.021"}],"arxiv_eprints":[{"value":"1207.7214"}],"publication_info":[{"journal_title":"Phys. Lett. B","year":2012}],"document_type":["article"]}}]}}`))
	}))
	defer server.Close()
	connector := NewInspireHEPConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "Higgs boson", Limit: 3})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if response.RawRef != "inspire-hep:/api/literature?q=Higgs+boson&size=3&sort=mostrecent" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "inspire-hep" {
		t.Fatalf("Source = %q", record.Source)
	}
	if record.SourceID != "1234567" {
		t.Fatalf("SourceID = %q", record.SourceID)
	}
	if record.Identifiers.DOI != "10.1016/j.physletb.2012.08.021" {
		t.Fatalf("DOI = %q", record.Identifiers.DOI)
	}
	if record.Identifiers.ArXivID != "1207.7214" {
		t.Fatalf("ArXivID = %q", record.Identifiers.ArXivID)
	}
	if record.Title != "Observation of the Higgs boson" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2012 {
		t.Fatalf("Year = %d", record.Year)
	}
	if record.Venue != "Phys. Lett. B" {
		t.Fatalf("Venue = %q", record.Venue)
	}
	if !record.OpenAccess {
		t.Fatal("OpenAccess = false, want true (has arXiv ID)")
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Identifiers.DOI != "10.1016/j.physletb.2012.08.021" {
		t.Fatalf("papers[0].DOI = %q", papers[0].Identifiers.DOI)
	}
}

func TestInspireHEPSearchEmptyMetadataGraceful(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"hits":{"total":1,"hits":[{"id":"9999","metadata":{"titles":[],"abstracts":[],"dois":[],"arxiv_eprints":[],"publication_info":[]}}]}}`))
	}))
	defer server.Close()
	connector := NewInspireHEPConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	if response.Records[0].Title != "" || response.Records[0].Year != 0 {
		t.Fatalf("expected empty fields for empty metadata: %+v", response.Records[0])
	}
}
