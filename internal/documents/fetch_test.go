package documents

import (
	"bytes"
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
	gitignorePath := filepath.Join(projectPath, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("*.tmp\n"), 0o600); err != nil {
		t.Fatalf("write prior .gitignore: %v", err)
	}
	documentDir := filepath.Join(projectPath, "documents", "open-access")
	if err := os.MkdirAll(documentDir, 0o755); err != nil {
		t.Fatalf("create document directory: %v", err)
	}
	priorPath := filepath.Join(documentDir, "10-1000-example.pdf")
	if err := os.WriteFile(priorPath, []byte("prior PDF\n"), 0o600); err != nil {
		t.Fatalf("write prior PDF: %v", err)
	}
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
	info, err := os.Stat(asset.LocalPath)
	if err != nil {
		t.Fatalf("stat PDF: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("PDF mode = %o, want 644", info.Mode().Perm())
	}
	entries, err := os.ReadDir(documentDir)
	if err != nil {
		t.Fatalf("read document directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(asset.LocalPath) {
		t.Fatalf("document directory entries = %#v, want only %s", entries, filepath.Base(asset.LocalPath))
	}
	gitignoreData, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if string(gitignoreData) != "*.tmp\ndocuments/\n" {
		t.Fatalf(".gitignore = %q, want prior content plus documents entry", gitignoreData)
	}
	gitignoreInfo, err := os.Stat(gitignorePath)
	if err != nil {
		t.Fatalf("stat .gitignore: %v", err)
	}
	if gitignoreInfo.Mode().Perm() != 0o600 {
		t.Fatalf(".gitignore mode = %o, want 600", gitignoreInfo.Mode().Perm())
	}
	projectEntries, err := os.ReadDir(projectPath)
	if err != nil {
		t.Fatalf("read project directory: %v", err)
	}
	if len(projectEntries) != 2 || projectEntries[0].Name() != ".gitignore" || projectEntries[1].Name() != "documents" {
		t.Fatalf("project directory entries = %#v, want only .gitignore and documents", projectEntries)
	}
}

func TestFetchPDFByDOIDoesNotWriteThroughSymlinkedDestination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4 redirected bytes"))
	}))
	defer server.Close()
	projectPath := t.TempDir()
	documentDir := filepath.Join(projectPath, "documents", "open-access")
	if err := os.MkdirAll(documentDir, 0o755); err != nil {
		t.Fatalf("create document directory: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.pdf")
	outsideBefore := []byte("outside document must remain unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside document: %v", err)
	}
	destPath := filepath.Join(documentDir, "10-1000-symlink.pdf")
	if err := os.Symlink(outsidePath, destPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	_, err := FetchPDFByDOI(context.Background(), projectPath, "10.1000/symlink", OpenAccessMetadata{OpenAccess: true, OAStatus: "gold", License: "cc-by", PDFURL: server.URL + "/paper.pdf"})
	if err == nil {
		t.Fatal("FetchPDFByDOI succeeded with a symlinked destination")
	}
	outsideAfter, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside document: %v", readErr)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("FetchPDFByDOI wrote through symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, lstatErr := os.Lstat(destPath)
	if lstatErr != nil {
		t.Fatalf("lstat destination: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("FetchPDFByDOI replaced symlink despite rejecting destination: mode=%v", info.Mode())
	}
}

func TestFetchPDFByDOIDoesNotWriteThroughSymlinkedGitignore(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF-1.4 fetched fixture"))
	}))
	defer server.Close()
	projectPath := t.TempDir()
	outsidePath := filepath.Join(t.TempDir(), "outside-ignore")
	outsideBefore := []byte("keep-outside-unchanged\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside ignore file: %v", err)
	}
	gitignorePath := filepath.Join(projectPath, ".gitignore")
	if err := os.Symlink(outsidePath, gitignorePath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	_, err := FetchPDFByDOI(context.Background(), projectPath, "10.1000/gitignore-symlink", OpenAccessMetadata{OpenAccess: true, OAStatus: "gold", License: "cc-by", PDFURL: server.URL + "/paper.pdf"})
	if err == nil {
		t.Fatal("FetchPDFByDOI succeeded with a symlinked .gitignore")
	}
	outsideAfter, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside ignore file: %v", readErr)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("FetchPDFByDOI wrote through .gitignore symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, lstatErr := os.Lstat(gitignorePath)
	if lstatErr != nil {
		t.Fatalf("lstat .gitignore: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("FetchPDFByDOI replaced .gitignore symlink despite rejecting it: mode=%v", info.Mode())
	}
	documentPath := filepath.Join(projectPath, "documents", "open-access", "10-1000-gitignore-symlink.pdf")
	if _, statErr := os.Stat(documentPath); !os.IsNotExist(statErr) {
		t.Fatalf("document path stat error = %v, want not exist", statErr)
	}
}
