package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestERICSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("search") != "reading comprehension" {
			t.Fatalf("search = %q", r.URL.Query().Get("search"))
		}
		if r.URL.Query().Get("rows") != "3" {
			t.Fatalf("rows = %q", r.URL.Query().Get("rows"))
		}
		if r.URL.Query().Get("format") != "json" {
			t.Fatalf("format = %q", r.URL.Query().Get("format"))
		}
		_, _ = w.Write([]byte(`{"response":{"numFound":12345,"docs":[{
			"id":"EJ1296630",
			"title":"The Science of Reading Comprehension Instruction",
			"author":["Duke, Nell K.","Pearson, P. David"],
			"description":"Decades of research on comprehension instruction.",
			"subject":["Reading Instruction","Reading Comprehension"],
			"publicationdateyear":2021,
			"peerreviewed":"T",
			"publicationtype":["Journal Articles","Reports - Descriptive"]
		}]}}`))
	}))
	defer server.Close()

	connector := NewERICConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "reading comprehension", Limit: 3})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if response.RawRef != "eric:/?search=reading+comprehension&rows=3" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "eric" {
		t.Fatalf("Source = %q", r.Source)
	}
	if r.SourceID != "EJ1296630" {
		t.Fatalf("SourceID = %q", r.SourceID)
	}
	if r.Identifiers.CrossrefID != "eric:EJ1296630" {
		t.Fatalf("CrossrefID = %q", r.Identifiers.CrossrefID)
	}
	if r.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty", r.Identifiers.DOI)
	}
	if r.Title != "The Science of Reading Comprehension Instruction" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Year != 2021 {
		t.Fatalf("Year = %d, want 2021", r.Year)
	}
	if r.Abstract != "Decades of research on comprehension instruction." {
		t.Fatalf("Abstract = %q", r.Abstract)
	}
	if len(r.URLs) != 1 || r.URLs[0] != "https://eric.ed.gov/?id=EJ1296630" {
		t.Fatalf("URLs = %v", r.URLs)
	}
	if r.Metadata["peerreviewed"] != "T" {
		t.Fatalf("peerreviewed = %q", r.Metadata["peerreviewed"])
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "The Science of Reading Comprehension Instruction" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}

func TestERICSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("rows") != "25" {
			t.Fatalf("default rows = %q, want 25", r.URL.Query().Get("rows"))
		}
		_, _ = w.Write([]byte(`{"response":{"numFound":0,"docs":[]}}`))
	}))
	defer server.Close()

	connector := NewERICConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}

func TestERICSearchFallbackURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":{"numFound":1,"docs":[{
			"id":"ED123456",
			"title":"No URL Document",
			"publicationdateyear":2010
		}]}}`))
	}))
	defer server.Close()

	connector := NewERICConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if len(r.URLs) != 1 || r.URLs[0] != "https://eric.ed.gov/?id=ED123456" {
		t.Fatalf("URLs = %v, want fallback eric.ed.gov URL", r.URLs)
	}
}
