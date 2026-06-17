package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNASAADSConnectorSearchesAndNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/search/query" || r.URL.Query().Get("q") != "10.1000/ads" || r.URL.Query().Get("rows") != "1" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"response":{"docs":[{"bibcode":"2024ApJ...123A...1B","title":["ADS fixture title"],"doi":["10.1000/ADS"],"year":"2024","pub":"Astrophysical Journal"}]}}`))
	}))
	defer server.Close()
	response, err := NewNASAADSConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(context.Background(), SourceQuery{Terms: "10.1000/ads", Limit: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 || response.Records[0].Source != "ads" || response.Records[0].SourceID != "2024ApJ...123A...1B" || response.Records[0].Identifiers.DOI != "10.1000/ads" || response.Records[0].Metadata["bibcode"] == "" {
		t.Fatalf("response = %#v", response)
	}
}
