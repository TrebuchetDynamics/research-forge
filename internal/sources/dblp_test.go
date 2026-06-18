package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDBLPSearchNormalizesCSRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/publ/api" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "attention mechanism" {
			t.Fatalf("q = %q", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("format") != "json" || r.URL.Query().Get("h") != "5" {
			t.Fatalf("query params = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"result":{"hits":{"@total":"1","hit":[{"@id":"123","info":{"title":"Attention is All You Need","authors":{"author":[{"text":"Ashish Vaswani"},{"text":"Noam Shazeer"}]},"year":"2017","venue":"NeurIPS","doi":"10.48550/arxiv.1706.03762","url":"https://dblp.org/rec/conf/nips/VaswaniS17","ee":"https://arxiv.org/abs/1706.03762"}}]}}}`))
	}))
	defer server.Close()
	connector := NewDBLPConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "attention mechanism", Limit: 5})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if response.RawRef != "dblp:/search/publ/api?f=0&format=json&h=5&q=attention+mechanism" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "dblp" {
		t.Fatalf("Source = %q", record.Source)
	}
	if record.SourceID != "123" {
		t.Fatalf("SourceID = %q", record.SourceID)
	}
	if record.Title != "Attention is All You Need" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2017 {
		t.Fatalf("Year = %d", record.Year)
	}
	if record.Venue != "NeurIPS" {
		t.Fatalf("Venue = %q", record.Venue)
	}
	if record.Identifiers.DOI != "10.48550/arxiv.1706.03762" {
		t.Fatalf("DOI = %q", record.Identifiers.DOI)
	}
	if record.Metadata["authors"] != "Ashish Vaswani; Noam Shazeer" {
		t.Fatalf("authors metadata = %q", record.Metadata["authors"])
	}
}

func TestDBLPSearchSingleAuthor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":{"hits":{"@total":"1","hit":[{"@id":"456","info":{"title":"Solo Paper","authors":{"author":{"text":"Jane Doe"}},"year":"2020","venue":"ICML","doi":"","url":"","ee":""}}]}}}`))
	}))
	defer server.Close()
	connector := NewDBLPConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "solo"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if response.Records[0].Metadata["authors"] != "Jane Doe" {
		t.Fatalf("single author = %q", response.Records[0].Metadata["authors"])
	}
}
