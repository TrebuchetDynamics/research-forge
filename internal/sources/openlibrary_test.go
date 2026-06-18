package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const openLibraryFixture = `{
  "numFound": 500,
  "start": 0,
  "docs": [
    {
      "key": "/works/OL20709638W",
      "title": "Photosynthesis and Energy Capture",
      "author_name": ["John Author", "Jane Contributor"],
      "first_publish_year": 2010,
      "isbn": ["9780123456789"],
      "publisher": ["MIT Press", "Other Press"],
      "subject": ["Photosynthesis", "Energy conversion"],
      "ebook_access": "public"
    },
    {
      "key": "/works/OL99999W",
      "title": "Climate Science for Everyone",
      "author_name": ["Alice Climate"],
      "first_publish_year": 2018,
      "isbn": [],
      "publisher": ["Oxford University Press"],
      "subject": ["Climate change"],
      "ebook_access": "no_ebook"
    }
  ]
}`

func TestOpenLibrarySearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search.json" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		if r.URL.Query().Get("q") == "" {
			t.Fatalf("missing q param")
		}
		_, _ = w.Write([]byte(openLibraryFixture))
	}))
	defer server.Close()

	connector := NewOpenLibraryConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	if connector.Name() != "openlibrary" {
		t.Fatalf("Name = %q, want openlibrary", connector.Name())
	}
	resp, err := connector.Search(context.Background(), SourceQuery{Terms: "photosynthesis", Limit: 25})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(resp.Records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(resp.Records))
	}
	r := resp.Records[0]
	if r.Source != "openlibrary" {
		t.Fatalf("Source = %q", r.Source)
	}
	if r.SourceID != "OL20709638W" {
		t.Fatalf("SourceID = %q, want OL20709638W", r.SourceID)
	}
	if r.Identifiers.CrossrefID != "openlibrary:OL20709638W" {
		t.Fatalf("CrossrefID = %q, want openlibrary:OL20709638W", r.Identifiers.CrossrefID)
	}
	if r.Year != 2010 {
		t.Fatalf("Year = %d, want 2010", r.Year)
	}
	if r.Title != "Photosynthesis and Energy Capture" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Publisher != "MIT Press" {
		t.Fatalf("Publisher = %q, want MIT Press", r.Publisher)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess should be true when ebook_access=public")
	}
	if r.Metadata["authors"] != "John Author; Jane Contributor" {
		t.Fatalf("authors = %q", r.Metadata["authors"])
	}
	if r.Metadata["isbn"] != "9780123456789" {
		t.Fatalf("isbn = %q", r.Metadata["isbn"])
	}
	if r.Metadata["ebook_access"] != "public" {
		t.Fatalf("ebook_access = %q", r.Metadata["ebook_access"])
	}
	papers, err := PaperRecords(resp)
	if err != nil {
		t.Fatalf("PaperRecords returned error: %v", err)
	}
	if len(papers) != 2 {
		t.Fatalf("papers = %d, want 2", len(papers))
	}
}

func TestOpenLibrarySearchPublicEbookIsOpenAccess(t *testing.T) {
	for _, tc := range []struct {
		access string
		wantOA bool
	}{
		{"public", true},
		{"borrowable", false},
		{"no_ebook", false},
		{"printdisabled", false},
	} {
		fixture := `{"numFound":1,"start":0,"docs":[{"key":"/works/OL1W","title":"Book","author_name":["X"],"first_publish_year":2000,"ebook_access":"` + tc.access + `"}]}`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(fixture))
		}))
		resp, err := NewOpenLibraryConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(
			context.Background(), SourceQuery{Terms: "test"})
		server.Close()
		if err != nil {
			t.Fatalf("%s: Search error: %v", tc.access, err)
		}
		if len(resp.Records) != 1 {
			t.Fatalf("%s: records = %d", tc.access, len(resp.Records))
		}
		if resp.Records[0].OpenAccess != tc.wantOA {
			t.Fatalf("%s: OpenAccess = %v, want %v", tc.access, resp.Records[0].OpenAccess, tc.wantOA)
		}
	}
}

func TestOpenLibrarySearchSkipsBlankTitles(t *testing.T) {
	fixture := `{"numFound":2,"start":0,"docs":[
		{"key":"/works/OL1W","title":"","author_name":["Author"],"first_publish_year":2000},
		{"key":"/works/OL2W","title":"Valid Title","author_name":["Author"],"first_publish_year":2001}
	]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fixture))
	}))
	defer server.Close()

	resp, err := NewOpenLibraryConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(
		context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(resp.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank title skipped)", len(resp.Records))
	}
}

func TestOpenLibrarySearchDefaultLimit(t *testing.T) {
	var gotLimit string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotLimit = r.URL.Query().Get("limit")
		_, _ = w.Write([]byte(`{"numFound":0,"start":0,"docs":[]}`))
	}))
	defer server.Close()

	_, _ = NewOpenLibraryConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(
		context.Background(), SourceQuery{Terms: "science", Limit: 0})
	if gotLimit != "25" {
		t.Fatalf("limit = %q, want 25", gotLimit)
	}
}
