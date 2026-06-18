package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOSFSearchNormalizesPreprintRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/preprints/" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("filter[title]") != "cognitive bias" {
			t.Fatalf("filter[title] = %q", r.URL.Query().Get("filter[title]"))
		}
		if r.URL.Query().Get("page[size]") != "3" {
			t.Fatalf("page[size] = %q", r.URL.Query().Get("page[size]"))
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"abc12","attributes":{"title":"Cognitive Biases in Decision Making","description":"We study cognitive biases.","date_published":"2024-01-15T00:00:00.000000","tags":["psychology","decision-making"],"preprint_doi":"10.31234/osf.io/abc12","license":{"name":"CC-By Attribution 4.0 International"}},"links":{"html":"https://osf.io/abc12/"}}],"links":{"next":null}}`))
	}))
	defer server.Close()
	connector := NewOSFConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "cognitive bias", Limit: 3})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if response.RawRef != "osf:/v2/preprints/?filter%5Btitle%5D=cognitive+bias&page%5Bsize%5D=3" {
		t.Fatalf("RawRef = %q", response.RawRef)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	record := response.Records[0]
	if record.Source != "osf" {
		t.Fatalf("Source = %q", record.Source)
	}
	if record.SourceID != "abc12" {
		t.Fatalf("SourceID = %q", record.SourceID)
	}
	if record.Identifiers.DOI != "10.31234/osf.io/abc12" {
		t.Fatalf("DOI = %q", record.Identifiers.DOI)
	}
	if record.Title != "Cognitive Biases in Decision Making" {
		t.Fatalf("Title = %q", record.Title)
	}
	if record.Year != 2024 {
		t.Fatalf("Year = %d", record.Year)
	}
	if !record.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if record.License != "CC-By Attribution 4.0 International" {
		t.Fatalf("License = %q", record.License)
	}
	if record.Metadata["tags"] != "psychology; decision-making" {
		t.Fatalf("tags = %q", record.Metadata["tags"])
	}
	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if papers[0].Title != "Cognitive Biases in Decision Making" {
		t.Fatalf("papers[0].Title = %q", papers[0].Title)
	}
}
