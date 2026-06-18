package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDataverseSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search" {
			t.Fatalf("path = %q, want /api/search", r.URL.Path)
		}
		if r.URL.Query().Get("type") != "dataset" {
			t.Fatalf("type = %q, want dataset", r.URL.Query().Get("type"))
		}
		_, _ = w.Write([]byte(`{
			"status": "OK",
			"data": {
				"q": "climate change",
				"total_count": 1,
				"items": [{
					"name": "Climate Helplessness Replication Data",
					"type": "dataset",
					"url": "https://doi.org/10.7910/DVN/TESTXX",
					"global_id": "doi:10.7910/DVN/TESTXX",
					"description": "Replication data for climate helplessness study.",
					"published_at": "2022-05-15T10:00:00Z",
					"publisher": "Social Psychology Dataverse",
					"subjects": ["Social Sciences", "Psychology"],
					"authors": ["Smith, Jane", "Doe, John"]
				}]
			}
		}`))
	}))
	defer server.Close()

	connector := NewDataverseConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "climate change", Limit: 5})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "dataverse" {
		t.Fatalf("Source = %q, want dataverse", r.Source)
	}
	if r.SourceID != "doi:10.7910/DVN/TESTXX" {
		t.Fatalf("SourceID = %q, want doi:10.7910/DVN/TESTXX", r.SourceID)
	}
	if r.Identifiers.DOI != "10.7910/dvn/testxx" {
		t.Fatalf("DOI = %q, want 10.7910/dvn/testxx", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty when DOI set", r.Identifiers.CrossrefID)
	}
	if r.Title != "Climate Helplessness Replication Data" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Year != 2022 {
		t.Fatalf("Year = %d, want 2022", r.Year)
	}
	if r.Abstract != "Replication data for climate helplessness study." {
		t.Fatalf("Abstract = %q", r.Abstract)
	}
	if r.Publisher != "Harvard Dataverse" {
		t.Fatalf("Publisher = %q, want Harvard Dataverse", r.Publisher)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if len(r.URLs) != 1 || r.URLs[0] != "https://doi.org/10.7910/DVN/TESTXX" {
		t.Fatalf("URLs = %v", r.URLs)
	}
	if r.Metadata["authors"] != "Smith, Jane; Doe, John" {
		t.Fatalf("authors = %q, want Smith, Jane; Doe, John", r.Metadata["authors"])
	}
	if r.Metadata["subjects"] != "Social Sciences; Psychology" {
		t.Fatalf("subjects = %q", r.Metadata["subjects"])
	}
	if r.Metadata["dataverse"] != "Social Psychology Dataverse" {
		t.Fatalf("dataverse = %q", r.Metadata["dataverse"])
	}

	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if len(papers) != 1 || papers[0].Title != "Climate Helplessness Replication Data" {
		t.Fatalf("papers round-trip failed")
	}
}

func TestDataverseSearchCrossrefIDFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"status": "OK",
			"data": {
				"total_count": 1,
				"items": [{
					"name": "No DOI Dataset",
					"type": "dataset",
					"url": "https://dataverse.example.com/dataset/123",
					"global_id": "hdl:12345/67890",
					"description": "A dataset without a DOI.",
					"published_at": "2021-01-01T00:00:00Z",
					"publisher": "Test Dataverse",
					"subjects": [],
					"authors": []
				}]
			}
		}`))
	}))
	defer server.Close()

	connector := NewDataverseConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	// global_id starts with "hdl:" not "doi:", so DOI should be empty and
	// CrossrefID should be "dataverse:<global_id>".
	if r.Identifiers.DOI != "" {
		t.Fatalf("DOI = %q, want empty for non-DOI global_id", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "dataverse:hdl:12345/67890" {
		t.Fatalf("CrossrefID = %q, want dataverse:hdl:12345/67890", r.Identifiers.CrossrefID)
	}
}

func TestDataverseSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"status": "OK",
			"data": {
				"total_count": 2,
				"items": [
					{"name": "", "type": "dataset", "url": "", "global_id": "doi:10.0/blank", "description": "", "published_at": "2021-01-01T00:00:00Z", "publisher": "", "subjects": [], "authors": []},
					{"name": "Valid Dataset", "type": "dataset", "url": "https://doi.org/10.0/valid", "global_id": "doi:10.0/valid", "description": "Valid.", "published_at": "2022-06-01T00:00:00Z", "publisher": "Test", "subjects": [], "authors": []}
				]
			}
		}`))
	}))
	defer server.Close()

	connector := NewDataverseConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank title skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid Dataset" {
		t.Fatalf("Title = %q, want Valid Dataset", response.Records[0].Title)
	}
}

func TestDataverseSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("per_page") != "25" {
			t.Fatalf("default per_page = %q, want 25", r.URL.Query().Get("per_page"))
		}
		_, _ = w.Write([]byte(`{"status":"OK","data":{"total_count":0,"items":[]}}`))
	}))
	defer server.Close()

	connector := NewDataverseConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "genetics", Limit: 0})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
