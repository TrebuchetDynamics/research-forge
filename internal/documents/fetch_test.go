package documents

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestFetchPDFByDOIRejectsOversizedContentLength(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Length", strconv.FormatInt(maxPDFDownloadBytes+1, 10))
		_, _ = w.Write([]byte("%PDF-1.4 too large"))
	}))
	defer server.Close()

	_, err := FetchPDFByDOI(context.Background(), t.TempDir(), "10.1000/oversized", OpenAccessMetadata{OpenAccess: true, OAStatus: "gold", License: "cc-by", PDFURL: server.URL + "/paper.pdf"})
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("FetchPDFByDOI error = %v, want too large", err)
	}
}

func TestFetchPDFByDOIUsesLegalURLAndMockHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/paper.pdf" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4 fetched fixture"))
	}))
	defer server.Close()
	projectPath := t.TempDir()
	asset, err := FetchPDFByDOI(context.Background(), projectPath, "10.1000/example", OpenAccessMetadata{OpenAccess: true, OAStatus: "gold", License: "cc-by", PDFURL: server.URL + "/paper.pdf"})
	if err != nil {
		t.Fatalf("FetchPDFByDOI returned error: %v", err)
	}
	if asset.PaperID != "10.1000/example" || asset.AcquisitionSource != "open-access-pdf" || asset.License != "cc-by" || asset.OAStatus != "gold" || asset.MIMEType != "application/pdf" {
		t.Fatalf("asset = %#v", asset)
	}
	if filepath.Dir(asset.LocalPath) != filepath.Join(projectPath, "documents", "open-access") {
		t.Fatalf("LocalPath = %q", asset.LocalPath)
	}
	data, err := os.ReadFile(asset.LocalPath)
	if err != nil {
		t.Fatalf("read PDF: %v", err)
	}
	if string(data) != "%PDF-1.4 fetched fixture" {
		t.Fatalf("data = %q", data)
	}
}
