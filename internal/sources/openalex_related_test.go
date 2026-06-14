package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAlexConnectorRelatedWorksDiscoversRelatedIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/works/W1" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"https://openalex.org/W1","related_works":["https://openalex.org/W2","https://openalex.org/W3"]}`))
	}))
	defer server.Close()

	response, err := NewOpenAlexConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).RelatedWorks(context.Background(), "https://openalex.org/W1", 1)
	if err != nil {
		t.Fatalf("RelatedWorks returned error: %v", err)
	}
	if response.RawRef != "openalex:/works/W1/related?limit=1" {
		t.Fatalf("rawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 || response.Records[0].Identifiers.OpenAlexID != "W2" || response.Records[0].Metadata["related_to"] != "W1" {
		t.Fatalf("records = %#v", response.Records)
	}
}
