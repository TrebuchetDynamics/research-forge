package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteParseRejectsOversizedPDFInput(t *testing.T) {
	pdfPath := filepath.Join(t.TempDir(), "oversized.pdf")
	file, err := os.Create(pdfPath)
	if err != nil {
		t.Fatalf("create pdf: %v", err)
	}
	if err := file.Truncate(maxParsePDFBytes + 1); err != nil {
		_ = file.Close()
		t.Fatalf("truncate pdf: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close pdf: %v", err)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"--project", t.TempDir(), "parse", "--paper", "paper-1", "--parser", "grobid", "--pdf", pdfPath}, stdout, stderr)
	if code == 0 {
		t.Fatalf("expected non-zero exit, stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "too large") {
		t.Fatalf("stderr = %s, want too large", stderr.String())
	}
}
