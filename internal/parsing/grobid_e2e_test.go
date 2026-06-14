package parsing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestOptInGROBIDRealEndpointParse parses a real PDF through a live GROBID
// endpoint, complementing the deterministic mock-TEI coverage. It is opt-in:
// set RFORGE_GROBID_E2E_URL to a running GROBID server and RFORGE_GROBID_E2E_PDF
// to a local PDF. It skips cleanly when either is unset, so the normal,
// network-free suite is unaffected.
func TestOptInGROBIDRealEndpointParse(t *testing.T) {
	baseURL := os.Getenv("RFORGE_GROBID_E2E_URL")
	if baseURL == "" {
		t.Skip("set RFORGE_GROBID_E2E_URL to a running GROBID endpoint to run this integration")
	}
	pdfPath := os.Getenv("RFORGE_GROBID_E2E_PDF")
	if pdfPath == "" {
		t.Skip("set RFORGE_GROBID_E2E_PDF to a local PDF to parse against the GROBID endpoint")
	}
	pdf, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Fatalf("read RFORGE_GROBID_E2E_PDF: %v", err)
	}

	client := NewGROBIDClient(GROBIDClientOptions{BaseURL: baseURL, Timeout: 60 * time.Second, Version: os.Getenv("RFORGE_GROBID_E2E_VERSION")})
	doc, err := client.Parse(context.Background(), pdf, ParseOptions{PaperID: "grobid-e2e"})
	if err != nil {
		t.Fatalf("real GROBID parse failed: %v", err)
	}
	if doc.ParserName != "grobid" || doc.PaperID != "grobid-e2e" {
		t.Fatalf("doc metadata = %#v", doc)
	}
	if strings.TrimSpace(doc.Title) == "" && len(doc.Sections) == 0 {
		t.Fatalf("expected GROBID to return a title or full-text sections, got %#v", doc)
	}
}

// TestGROBIDClientErrorsOnUnreachableEndpoint verifies the parser surfaces a
// transport error (rather than panicking or hanging) when the GROBID endpoint
// cannot be reached. This runs in the normal suite without any external server.
func TestGROBIDClientErrorsOnUnreachableEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	baseURL := server.URL
	server.Close() // the endpoint is now closed, so requests must fail

	client := NewGROBIDClient(GROBIDClientOptions{BaseURL: baseURL, Timeout: time.Second, Version: "test"})
	if _, err := client.Parse(context.Background(), []byte("%PDF-1.4 fixture"), ParseOptions{PaperID: "paper-1"}); err == nil {
		t.Fatalf("expected an error parsing against an unreachable GROBID endpoint")
	}
}
