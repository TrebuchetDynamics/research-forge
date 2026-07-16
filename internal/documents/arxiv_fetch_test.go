package documents

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestFetchArXivAssetDoesNotWriteThroughSymlinkedDestination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("%PDF redirected arXiv bytes"))
	}))
	defer server.Close()
	projectPath := t.TempDir()
	documentDir := filepath.Join(projectPath, "documents", "arxiv")
	if err := os.MkdirAll(documentDir, 0o755); err != nil {
		t.Fatalf("create arXiv document directory: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.pdf")
	outsideBefore := []byte("outside arXiv document must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside arXiv document: %v", err)
	}
	destPath := filepath.Join(documentDir, "2401-00001.pdf")
	if err := os.Symlink(outsidePath, destPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	_, err := FetchArXivAsset(context.Background(), projectPath, "2401.00001", server.URL+"/paper.pdf", "pdf")
	if err == nil {
		t.Fatal("FetchArXivAsset succeeded with a symlinked destination")
	}
	outsideAfter, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside arXiv document: %v", readErr)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("FetchArXivAsset wrote through symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, lstatErr := os.Lstat(destPath)
	if lstatErr != nil {
		t.Fatalf("lstat arXiv destination: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("FetchArXivAsset replaced symlink despite rejecting destination: mode=%v", info.Mode())
	}
}
