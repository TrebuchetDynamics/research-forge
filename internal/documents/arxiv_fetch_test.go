package documents

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestFetchArXivAssetDownloadsPDFAndSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/pdf/2401.00001":
			_, _ = w.Write([]byte("%PDF arxiv fixture"))
		case "/e-print/2401.00001":
			_, _ = w.Write([]byte("tex source fixture"))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	project := t.TempDir()

	pdf, err := FetchArXivAsset(context.Background(), project, "2401.00001", server.URL+"/pdf/2401.00001", "pdf")
	if err != nil {
		t.Fatalf("FetchArXivAsset pdf returned error: %v", err)
	}
	if pdf.AcquisitionSource != "arxiv-pdf" || pdf.MIMEType != "application/pdf" || pdf.LocalOnly {
		t.Fatalf("pdf asset = %#v", pdf)
	}
	if data, err := os.ReadFile(pdf.LocalPath); err != nil || !strings.Contains(string(data), "%PDF") {
		t.Fatalf("pdf file err=%v data=%s", err, data)
	}

	source, err := FetchArXivAsset(context.Background(), project, "2401.00001", server.URL+"/e-print/2401.00001", "source")
	if err != nil {
		t.Fatalf("FetchArXivAsset source returned error: %v", err)
	}
	if source.AcquisitionSource != "arxiv-source" || source.MIMEType != "application/gzip" || !source.LocalOnly {
		t.Fatalf("source asset = %#v", source)
	}
}
