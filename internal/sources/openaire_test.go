package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAIRESearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/publications" {
			t.Fatalf("path = %s, want /search/publications", r.URL.Path)
		}
		if r.URL.Query().Get("keywords") != "deep learning" {
			t.Fatalf("keywords = %q, want deep learning", r.URL.Query().Get("keywords"))
		}
		if r.URL.Query().Get("size") != "5" {
			t.Fatalf("size = %q, want 5", r.URL.Query().Get("size"))
		}
		if r.URL.Query().Get("format") != "json" {
			t.Fatalf("format = %q, want json", r.URL.Query().Get("format"))
		}
		if r.URL.Query().Get("page") != "1" {
			t.Fatalf("page = %q, want 1", r.URL.Query().Get("page"))
		}
		_, _ = w.Write([]byte(`{
			"response": {
				"results": {
					"result": [{
						"header": {
							"dri:objIdentifier": {"$": "doi_dedup___::4807efad8ff855adaa51d3c5c5390481"}
						},
						"metadata": {
							"oaf:entity": {
								"oaf:result": {
									"title": [
										{"@classid": "main title", "@classname": "main title", "@schemeid": "dnet:dataCite_title", "@schemename": "dnet:dataCite_title", "$": "Deep Learning Approaches"},
										{"@classid": "alternative title", "@classname": "alternative title", "@schemeid": "dnet:dataCite_title", "@schemename": "dnet:dataCite_title", "$": "deep learning approaches"}
									],
									"pid": {"@classid": "doi", "@classname": "Digital Object Identifier", "@schemeid": "dnet:pid_types", "@schemename": "dnet:pid_types", "$": "10.1007/978-3-031-47508-5_2"},
									"dateofacceptance": {"$": "2024-01-01"},
									"description": {"$": "A comprehensive overview of deep learning."},
									"bestaccessright": {"@classid": "OPEN", "@classname": "Open Access", "@schemeid": "dnet:access_modes", "@schemename": "dnet:access_modes"},
									"journal": {"$": "Topics in Catalysis"}
								}
							}
						}
					}]
				}
			}
		}`))
	}))
	defer server.Close()

	connector := NewOpenAIREConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "deep learning", Limit: 5})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "openaire" {
		t.Fatalf("Source = %q, want openaire", record.Source)
	}
	if record.SourceID != "doi_dedup___::4807efad8ff855adaa51d3c5c5390481" {
		t.Fatalf("SourceID = %q", record.SourceID)
	}
	if record.Title != "Deep Learning Approaches" {
		t.Fatalf("Title = %q, want Deep Learning Approaches", record.Title)
	}
	if record.Identifiers.DOI != "10.1007/978-3-031-47508-5_2" {
		t.Fatalf("DOI = %q, want 10.1007/978-3-031-47508-5_2", record.Identifiers.DOI)
	}
	if record.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty (DOI present)", record.Identifiers.CrossrefID)
	}
	if record.Year != 2024 {
		t.Fatalf("Year = %d, want 2024", record.Year)
	}
	if record.Abstract != "A comprehensive overview of deep learning." {
		t.Fatalf("Abstract = %q", record.Abstract)
	}
	if record.Venue != "Topics in Catalysis" {
		t.Fatalf("Venue = %q, want Topics in Catalysis", record.Venue)
	}
	if !record.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if len(record.URLs) != 1 || record.URLs[0] != "https://doi.org/10.1007/978-3-031-47508-5_2" {
		t.Fatalf("URLs = %v", record.URLs)
	}
}

func TestOpenAIRESearchCrossrefIDFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"response": {
				"results": {
					"result": [{
						"header": {
							"dri:objIdentifier": {"$": "openaire_dedup___::abc123"}
						},
						"metadata": {
							"oaf:entity": {
								"oaf:result": {
									"title": [{"@classid": "main title", "@classname": "main title", "@schemeid": "dnet:dataCite_title", "@schemename": "dnet:dataCite_title", "$": "No DOI Paper"}],
									"dateofacceptance": {"$": "2022-06-15"},
									"bestaccessright": {"@classid": "CLOSED", "@classname": "Closed Access", "@schemeid": "dnet:access_modes", "@schemename": "dnet:access_modes"}
								}
							}
						}
					}]
				}
			}
		}`))
	}))
	defer server.Close()

	connector := NewOpenAIREConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "no doi"})
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
	if record.Identifiers.CrossrefID != "openaire:openaire_dedup___::abc123" {
		t.Fatalf("CrossrefID = %q, want openaire:openaire_dedup___::abc123", record.Identifiers.CrossrefID)
	}
}

func TestOpenAIRESearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"response": {
				"results": {
					"result": [
						{
							"header": {"dri:objIdentifier": {"$": "id1"}},
							"metadata": {
								"oaf:entity": {
									"oaf:result": {
										"title": [{"@classid": "main title", "@classname": "main title", "@schemeid": "dnet:dataCite_title", "@schemename": "dnet:dataCite_title", "$": "   "}]
									}
								}
							}
						},
						{
							"header": {"dri:objIdentifier": {"$": "id2"}},
							"metadata": {
								"oaf:entity": {
									"oaf:result": {
										"title": [{"@classid": "main title", "@classname": "main title", "@schemeid": "dnet:dataCite_title", "@schemename": "dnet:dataCite_title", "$": "Valid Title"}]
									}
								}
							}
						}
					]
				}
			}
		}`))
	}))
	defer server.Close()

	connector := NewOpenAIREConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank-title record skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid Title" {
		t.Fatalf("Title = %q, want Valid Title", response.Records[0].Title)
	}
}

func TestOpenAIRESearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("size") != "25" {
			t.Fatalf("default size = %q, want 25", r.URL.Query().Get("size"))
		}
		if r.URL.Query().Get("format") != "json" {
			t.Fatalf("format = %q, want json", r.URL.Query().Get("format"))
		}
		if r.URL.Query().Get("page") != "1" {
			t.Fatalf("page = %q, want 1", r.URL.Query().Get("page"))
		}
		_, _ = w.Write([]byte(`{"response":{"results":{"result":[]}}}`))
	}))
	defer server.Close()

	connector := NewOpenAIREConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "test", Limit: 0})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
