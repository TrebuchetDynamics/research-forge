package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestZbMATHSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/document/_search" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("search_string") != "elliptic curves" {
			t.Fatalf("search_string = %q", r.URL.Query().Get("search_string"))
		}
		if r.URL.Query().Get("per_page") != "2" {
			t.Fatalf("per_page = %q", r.URL.Query().Get("per_page"))
		}
		// Real API structure: result is a flat array; title/venue are nested structs
		_, _ = w.Write([]byte(`{"result":[{
			"id": 7789123,
			"identifier": "07789123",
			"title": {"title": "Elliptic Curves and Modular Forms"},
			"year": 2023,
			"zbmath_url": "https://zbmath.org/7789123",
			"links": [{"identifier": "10.4007/annals.2023.100", "type": "doi", "url": "https://doi.org/10.4007/annals.2023.100"}],
			"source": {"series": [{"title": "Annals of Mathematics"}]},
			"msc": [{"code": "11G05"}, {"code": "14H52"}]
		}]}`))
	}))
	defer server.Close()

	connector := NewZbMATHConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "elliptic curves", Limit: 2})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	wantRawRef := "zbmath:/v1/document/_search?search_string=elliptic+curves&per_page=2"
	if response.RawRef != wantRawRef {
		t.Fatalf("RawRef = %q, want %q", response.RawRef, wantRawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "zbmath" {
		t.Fatalf("Source = %q", record.Source)
	}
	if record.SourceID != "7789123" {
		t.Fatalf("SourceID = %q", record.SourceID)
	}
	if record.Identifiers.DOI != "10.4007/annals.2023.100" {
		t.Fatalf("DOI = %q", record.Identifiers.DOI)
	}
	if record.Title != "Elliptic Curves and Modular Forms" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2023 {
		t.Fatalf("Year = %d, want 2023", record.Year)
	}
	if record.Venue != "Annals of Mathematics" {
		t.Fatalf("Venue = %q", record.Venue)
	}
	if len(record.URLs) != 1 || record.URLs[0] != "https://zbmath.org/7789123" {
		t.Fatalf("URLs = %v", record.URLs)
	}
	if record.Metadata["msc_codes"] != "11G05; 14H52" {
		t.Fatalf("msc_codes = %q", record.Metadata["msc_codes"])
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "Elliptic Curves and Modular Forms" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestZbMATHSearchSkipsRedactedTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":[
			{"id":1,"identifier":"00000001","title":{"title":"zbMATH Open Web Interface contents unavailable due to conflicting licenses."},"year":2020,"zbmath_url":"https://zbmath.org/1","links":[],"source":{},"msc":[]},
			{"id":2,"identifier":"00000002","title":{"title":"A Real Math Paper"},"year":2021,"zbmath_url":"https://zbmath.org/2","links":[{"identifier":"10.1234/real","type":"doi","url":""}],"source":{},"msc":[]}
		]}`))
	}))
	defer server.Close()

	connector := NewZbMATHConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "math"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (redacted title skipped)", len(response.Records))
	}
	if response.Records[0].Title != "A Real Math Paper" {
		t.Fatalf("Title = %q", response.Records[0].Title)
	}
}

func TestZbMATHSearchZblIDFallbackIdentifier(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":[{
			"id":1234567,
			"identifier":"01234567",
			"title":{"title":"No DOI Paper"},
			"year":2019,
			"zbmath_url":"https://zbmath.org/1234567",
			"links":[],
			"source":{},
			"msc":[]
		}]}`))
	}))
	defer server.Close()

	connector := NewZbMATHConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "euler"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	record := response.Records[0]
	if record.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty", record.Identifiers.DOI)
	}
	if record.Identifiers.CrossrefID != "zbmath:01234567" {
		t.Fatalf("CrossrefID = %q", record.Identifiers.CrossrefID)
	}
}

func TestZbMATHSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("per_page") != "25" {
			t.Fatalf("default per_page = %q, want 25", r.URL.Query().Get("per_page"))
		}
		_, _ = w.Write([]byte(`{"result":[]}`))
	}))
	defer server.Close()

	connector := NewZbMATHConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
