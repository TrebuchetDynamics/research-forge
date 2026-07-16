package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutePDFFetchDoesNotPartiallyReplaceDerivatives(t *testing.T) {
	bin := t.TempDir()
	pdftotext := filepath.Join(bin, "pdftotext")
	if err := os.WriteFile(pdftotext, []byte("#!/bin/sh\nprintf 'new extracted text\\n'\n"), 0o755); err != nil {
		t.Fatalf("write fake pdftotext: %v", err)
	}
	pdfimages := filepath.Join(bin, "pdfimages")
	if err := os.WriteFile(pdfimages, []byte("#!/bin/sh\nprintf 'partial image' > \"$3-000.png\"\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write failing pdfimages: %v", err)
	}
	t.Setenv("RFORGE_PDFTOTEXT_CMD", pdftotext)
	t.Setenv("RFORGE_PDFIMAGES_CMD", pdfimages)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("%PDF-1.4 derivative fixture"))
	}))
	defer server.Close()

	project := t.TempDir()
	textPath := filepath.Join(project, "documents", "text", "10-1000-example.txt")
	imagePath := filepath.Join(project, "documents", "images", "10-1000-example", "image-000.png")
	if err := os.MkdirAll(filepath.Dir(textPath), 0o755); err != nil {
		t.Fatalf("create text directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(imagePath), 0o755); err != nil {
		t.Fatalf("create image directory: %v", err)
	}
	textBefore := []byte("prior extracted text\n")
	imageBefore := []byte("prior image")
	if err := os.WriteFile(textPath, textBefore, 0o600); err != nil {
		t.Fatalf("write prior text: %v", err)
	}
	if err := os.WriteFile(imagePath, imageBefore, 0o600); err != nil {
		t.Fatalf("write prior image: %v", err)
	}
	var stdout, stderr bytes.Buffer

	code := Execute([]string{
		"--json", "--project", project, "pdf", "fetch",
		"--doi", "10.1000/example", "--pdf-url", server.URL + "/paper.pdf",
		"--license", "cc-by", "--oa-status", "gold",
	}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stdout.String(), "pdf_derivatives_failed") {
		t.Fatalf("code=%d stdout=%s stderr=%s, want derivative failure", code, stdout.String(), stderr.String())
	}
	for path, before := range map[string][]byte{textPath: textBefore, imagePath: imageBefore} {
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read preserved derivative %s: %v", path, err)
		}
		if !bytes.Equal(after, before) {
			t.Errorf("derivative %s changed:\n got: %s\nwant: %s", path, after, before)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat preserved derivative %s: %v", path, err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Errorf("derivative %s mode = %o, want 600", path, got)
		}
	}
	entries, err := os.ReadDir(filepath.Join(project, "documents", "images"))
	if err != nil {
		t.Fatalf("read image root: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "10-1000-example" {
		t.Fatalf("image root contains transaction debris: %v", entries)
	}
}
