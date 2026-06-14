package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAlexConnectorSearchAuthors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/authors" || r.URL.Query().Get("search") != "Ada Lovelace" || r.URL.Query().Get("per-page") != "1" {
			t.Fatalf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/A123","display_name":"Ada Lovelace","works_count":42}]}`))
	}))
	defer server.Close()

	entities, rawRef, err := NewOpenAlexConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).SearchAuthors(context.Background(), SourceQuery{Terms: "Ada Lovelace", Limit: 1})
	if err != nil {
		t.Fatalf("SearchAuthors returned error: %v", err)
	}
	if rawRef != "openalex:/authors?per-page=1&search=Ada+Lovelace" {
		t.Fatalf("rawRef = %q", rawRef)
	}
	if len(entities) != 1 || entities[0].Source != "openalex-author" || entities[0].SourceID != "A123" || entities[0].WorksCount != 42 {
		t.Fatalf("entities = %#v", entities)
	}
}

func TestOpenAlexConnectorSearchInstitutions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/institutions" || r.URL.Query().Get("search") != "University" {
			t.Fatalf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"results":[{"id":"https://openalex.org/I123","display_name":"Example University","works_count":100,"ror":"https://ror.org/123","country_code":"GB"}]}`))
	}))
	defer server.Close()

	entities, _, err := NewOpenAlexConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).SearchInstitutions(context.Background(), SourceQuery{Terms: "University"})
	if err != nil {
		t.Fatalf("SearchInstitutions returned error: %v", err)
	}
	if len(entities) != 1 || entities[0].Source != "openalex-institution" || entities[0].Metadata["ror"] != "https://ror.org/123" || entities[0].Metadata["country_code"] != "GB" {
		t.Fatalf("entities = %#v", entities)
	}
}
