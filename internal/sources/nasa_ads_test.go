package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNASAADSConnectorSearchesAndNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/search/query" || r.URL.Query().Get("q") != "10.1000/ads" || r.URL.Query().Get("rows") != "1" || r.URL.Query().Get("fl") != "bibcode,title,doi,year,author,pub,abstract,doctype,database" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"response":{"docs":[{"bibcode":"2024ApJ...123A...1B","title":["ADS fixture title"],"doi":["10.1000/ADS"],"year":"2024","pub":"Astrophysical Journal","abstract":"Physics astronomy abstract","doctype":"article","database":["astronomy"]}]}}`))
	}))
	defer server.Close()
	response, err := NewNASAADSConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL})).Search(context.Background(), SourceQuery{Terms: "10.1000/ads", Limit: 1})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(response.Records) != 1 || response.Records[0].Source != "ads" || response.Records[0].SourceID != "2024ApJ...123A...1B" || response.Records[0].Identifiers.DOI != "10.1000/ads" || response.Records[0].Identifiers.ADSBibcode == "" || response.Records[0].Abstract == "" || response.Records[0].Metadata["bibcode"] == "" || response.Records[0].Metadata["database"] != "astronomy" {
		t.Fatalf("response = %#v", response)
	}
}

func TestNASAADSConnectorExpandsCitationGraphAndRedactsToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Fatalf("authorization header = %q", got)
		}
		if r.URL.Query().Get("q") != "bibcode:2024ApJ...123A...1B" || r.URL.Query().Get("fl") != "bibcode,title,reference,citation" {
			t.Fatalf("unexpected graph request: %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"response":{"docs":[{"bibcode":"2024ApJ...123A...1B","title":["Seed"],"reference":["2020ApJ...000R...1A"],"citation":["2025ApJ...999C...1Z"]}]}}`))
	}))
	defer server.Close()
	connector := NewNASAADSConnector(NewNASAADSHTTPClient(server.URL, "secret-token"))
	expansion, err := connector.ExpandCitationGraph(context.Background(), "2024ApJ...123A...1B", 2)
	if err != nil {
		t.Fatalf("ExpandCitationGraph: %v", err)
	}
	if len(expansion.Edges) != 2 || expansion.RawRef != "ads:/v1/search/query?q=bibcode:2024ApJ...123A...1B" {
		t.Fatalf("expansion = %#v", expansion)
	}
	if RedactNASAADSToken("Authorization: Bearer secret-token") != "Authorization: Bearer [REDACTED_ADS_TOKEN]" {
		t.Fatalf("token not redacted")
	}
}
