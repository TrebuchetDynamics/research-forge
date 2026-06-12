package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUnpaywallConnectorLooksUpDOIAndNormalizesOpenAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/v2/10.5555%2Fexample" {
			t.Fatalf("path = %q, want escaped DOI path", r.URL.EscapedPath())
		}
		if r.URL.Query().Get("email") != "researcher@example.org" {
			t.Fatalf("email query = %q", r.URL.Query().Get("email"))
		}
		_, _ = w.Write([]byte(`{
			"doi": "10.5555/EXAMPLE",
			"is_oa": true,
			"oa_status": "gold",
			"best_oa_location": {
				"url": "https://example.org/article",
				"url_for_pdf": "https://example.org/article.pdf",
				"license": "cc-by",
				"host_type": "publisher"
			}
		}`))
	}))
	defer server.Close()

	connector := NewUnpaywallConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, UserAgent: "ResearchForge/test", Timeout: time.Second}), "researcher@example.org")
	record, err := connector.LookupDOI(context.Background(), "10.5555/example")
	if err != nil {
		t.Fatalf("LookupDOI returned error: %v", err)
	}
	if connector.Name() != "unpaywall" {
		t.Fatalf("Name = %q", connector.Name())
	}
	if record.DOI != "10.5555/example" || !record.OpenAccess || record.OAStatus != "gold" {
		t.Fatalf("record status = %#v", record)
	}
	if record.License != "cc-by" || record.BestURL != "https://example.org/article" || record.PDFURL != "https://example.org/article.pdf" {
		t.Fatalf("record location = %#v", record)
	}
	if record.SourceRef.Source != "unpaywall" || record.SourceRef.RawPayloadRef != "unpaywall:/v2/10.5555%2Fexample" {
		t.Fatalf("source ref = %#v", record.SourceRef)
	}
	if record.SourceRef.Metadata["host_type"] != "publisher" || record.SourceRef.Metadata["oa_status"] != "gold" {
		t.Fatalf("metadata = %#v", record.SourceRef.Metadata)
	}
}

func TestUnpaywallConnectorRequiresEmailWithoutLeakingConfiguredEmail(t *testing.T) {
	connector := NewUnpaywallConnector(NewHTTPClient(HTTPClientOptions{BaseURL: "http://127.0.0.1", UserAgent: "ResearchForge/test", Timeout: time.Second}), "")
	_, err := connector.LookupDOI(context.Background(), "10.5555/example")
	if err == nil {
		t.Fatalf("LookupDOI returned nil error without email")
	}
	if err.Error() != "unpaywall email is required" {
		t.Fatalf("error = %q", err.Error())
	}
}
