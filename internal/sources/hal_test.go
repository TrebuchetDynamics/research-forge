package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHALSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "neural networks" {
			t.Fatalf("q = %q", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("rows") != "2" {
			t.Fatalf("rows = %q", r.URL.Query().Get("rows"))
		}
		if r.URL.Query().Get("wt") != "json" {
			t.Fatalf("wt = %q", r.URL.Query().Get("wt"))
		}
		_, _ = w.Write([]byte(`{"response":{"numFound":37000,"docs":[{
			"docid":"4987097",
			"halId_s":"hal-04987097",
			"title_s":["Advancing Cybersecurity through Machine Learning"],
			"authFullName_s":["Smith, John","Brown, Jane"],
			"producedDateY_i":2025,
			"abstract_s":["Abstract text here."],
			"doiId_s":"10.9734/ajrcos/2025/v18i2572",
			"journalTitle_s":"Asian Journal of Research in Computer Science",
			"publisher_s":"Science Publishing Group",
			"openAccess_bool":true,
			"uri_s":"https://hal.science/hal-04987097v1"
		}]}}`))
	}))
	defer server.Close()

	connector := NewHALConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "neural networks", Limit: 2})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if response.RawRef != "hal:/search/?q=neural+networks&rows=2" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "hal" {
		t.Fatalf("Source = %q", r.Source)
	}
	if r.SourceID != "hal-04987097" {
		t.Fatalf("SourceID = %q", r.SourceID)
	}
	if r.Identifiers.DOI != "10.9734/ajrcos/2025/v18i2572" {
		t.Fatalf("DOI = %q", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty (DOI present)", r.Identifiers.CrossrefID)
	}
	if r.Title != "Advancing Cybersecurity through Machine Learning" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Year != 2025 {
		t.Fatalf("Year = %d, want 2025", r.Year)
	}
	if r.Abstract != "Abstract text here." {
		t.Fatalf("Abstract = %q", r.Abstract)
	}
	if r.Venue != "Asian Journal of Research in Computer Science" {
		t.Fatalf("Venue = %q", r.Venue)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if len(r.URLs) != 1 || r.URLs[0] != "https://hal.science/hal-04987097v1" {
		t.Fatalf("URLs = %v", r.URLs)
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "Advancing Cybersecurity through Machine Learning" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestHALSearchFallbackIdentifier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":{"numFound":1,"docs":[{
			"docid":"1234567",
			"halId_s":"hal-01234567",
			"title_s":["No DOI Paper"],
			"producedDateY_i":2019,
			"openAccess_bool":false,
			"uri_s":"https://hal.science/hal-01234567"
		}]}}`))
	}))
	defer server.Close()

	connector := NewHALConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	r := response.Records[0]
	if r.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "hal:hal-01234567" {
		t.Fatalf("CrossrefID = %q, want hal:hal-01234567", r.Identifiers.CrossrefID)
	}
}

func TestHALSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("rows") != "25" {
			t.Fatalf("default rows = %q, want 25", r.URL.Query().Get("rows"))
		}
		_, _ = w.Write([]byte(`{"response":{"numFound":0,"docs":[]}}`))
	}))
	defer server.Close()

	connector := NewHALConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
