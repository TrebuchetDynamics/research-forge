package sources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLensSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scholarly/search" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var req lensSearchRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if req.Query.QueryString.Query != "cancer immunotherapy" {
			t.Fatalf("query = %q, want cancer immunotherapy", req.Query.QueryString.Query)
		}
		if req.Size != 4 {
			t.Fatalf("size = %d, want 4", req.Size)
		}
		_, _ = w.Write([]byte(`{"total":1,"data":[{"lens_id":"000-000-000-000-001","title":"Cancer immunotherapy advances","abstract":"We review...","year_published":2024,"scholarly_citations_count":42,"authors":[{"first_name":"John","last_name":"Smith","ids":[{"type":"orcid","value":"0000-0001-2345-6789"}]}],"open_access":{"is_oa":true,"color":"gold"},"external_ids":[{"type":"doi","value":"10.1234/test"},{"type":"arxiv","value":"2401.12345"}],"source":{"title":"Nature Medicine","publisher":"Springer Nature"}}]}`))
	}))
	defer server.Close()
	connector := NewLensConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "cancer immunotherapy", Limit: 4})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "lens" {
		t.Fatalf("Source = %q", record.Source)
	}
	if record.SourceID != "000-000-000-000-001" {
		t.Fatalf("SourceID = %q", record.SourceID)
	}
	if record.Identifiers.DOI != "10.1234/test" {
		t.Fatalf("DOI = %q, want 10.1234/test", record.Identifiers.DOI)
	}
	if record.Identifiers.ArXivID != "2401.12345" {
		t.Fatalf("ArXivID = %q, want 2401.12345", record.Identifiers.ArXivID)
	}
	if record.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty (DOI present)", record.Identifiers.CrossrefID)
	}
	if record.Title != "Cancer immunotherapy advances" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2024 {
		t.Fatalf("Year = %d, want 2024", record.Year)
	}
	if !record.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if record.Venue != "Nature Medicine" {
		t.Fatalf("Venue = %q, want Nature Medicine", record.Venue)
	}
	if record.Publisher != "Springer Nature" {
		t.Fatalf("Publisher = %q, want Springer Nature", record.Publisher)
	}
	if record.Metadata["citations"] != "42" {
		t.Fatalf("citations metadata = %q, want 42", record.Metadata["citations"])
	}
	if record.Metadata["lens_id"] != "000-000-000-000-001" {
		t.Fatalf("lens_id metadata = %q", record.Metadata["lens_id"])
	}
}

func TestLensSearchNoIDFallsBackToCrossrefID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"total":1,"data":[{"lens_id":"999-999-999-999-001","title":"Obscure Paper","abstract":"","year_published":2020,"scholarly_citations_count":0,"authors":[],"open_access":{"is_oa":false,"color":""},"external_ids":[],"source":{"title":"","publisher":""}}]}`))
	}))
	defer server.Close()
	connector := NewLensConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "obscure"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty", record.Identifiers.DOI)
	}
	if record.Identifiers.ArXivID != "" {
		t.Fatalf("ArXivID = %q, want empty", record.Identifiers.ArXivID)
	}
	if record.Identifiers.CrossrefID != "lens:999-999-999-999-001" {
		t.Fatalf("CrossrefID = %q, want lens:999-999-999-999-001", record.Identifiers.CrossrefID)
	}
}

func TestLensSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var req lensSearchRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if req.Size != 25 {
			t.Fatalf("default size = %d, want 25", req.Size)
		}
		_, _ = w.Write([]byte(`{"total":0,"data":[]}`))
	}))
	defer server.Close()
	connector := NewLensConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
