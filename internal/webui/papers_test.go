package webui

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleParsedDoc = `{
  "SchemaVersion": "1",
  "PaperID": "10.1000/ap",
  "ParserName": "grobid",
  "Title": "Artificial Photosynthesis Review",
  "Authors": [{"Given": "Ada", "Family": "Lovelace"}],
  "Abstract": "We review water-splitting catalysts.",
  "Sections": [
    {"ID": "s1", "Title": "Introduction", "Passages": [
      {"ID": "p1", "SectionID": "s1", "Text": "Photosynthesis converts sunlight."}
    ]}
  ]
}`

func writeParsedDoc(t *testing.T, projectPath, stem, body string) {
	t.Helper()
	dir := filepath.Join(projectPath, "parsed")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir parsed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, stem+".json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write parsed doc: %v", err)
	}
}

func writeLocalPDF(t *testing.T, projectPath, stem string) {
	t.Helper()
	dir := filepath.Join(projectPath, "documents", "open-access")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir documents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, stem+".pdf"), []byte("%PDF-1.4 minimal"), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
}

func TestBuildPaperListReadsParsedDocuments(t *testing.T) {
	dir := t.TempDir()
	writeParsedDoc(t, dir, "10-1000-ap", sampleParsedDoc)

	vm, err := BuildPaperList(dir)
	if err != nil {
		t.Fatalf("BuildPaperList: %v", err)
	}
	if vm.Empty || len(vm.Papers) != 1 {
		t.Fatalf("paper list = %+v, want 1 paper", vm)
	}
	p := vm.Papers[0]
	if p.ID != "10-1000-ap" || p.Title != "Artificial Photosynthesis Review" {
		t.Fatalf("paper summary = %+v", p)
	}
	if p.PassageCount != 1 {
		t.Fatalf("passage count = %d, want 1", p.PassageCount)
	}
}

func TestPapersRoutesServeListAndDetail(t *testing.T) {
	dir := t.TempDir()
	writeParsedDoc(t, dir, "10-1000-ap", sampleParsedDoc)
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()

	listBody, listStatus, _ := getURL(t, ts.URL+"/papers")
	if listStatus != http.StatusOK {
		t.Fatalf("GET /papers status = %d", listStatus)
	}
	if !strings.Contains(listBody, "Artificial Photosynthesis Review") {
		t.Fatalf("/papers missing title: %s", listBody)
	}
	if !strings.Contains(listBody, "/papers/10-1000-ap") {
		t.Fatalf("/papers missing detail link: %s", listBody)
	}

	detailBody, detailStatus, _ := getURL(t, ts.URL+"/papers/10-1000-ap")
	if detailStatus != http.StatusOK {
		t.Fatalf("GET /papers/{id} status = %d", detailStatus)
	}
	for _, want := range []string{"We review water-splitting catalysts.", "Introduction", "Photosynthesis converts sunlight."} {
		if !strings.Contains(detailBody, want) {
			t.Fatalf("paper detail missing %q: %s", want, detailBody)
		}
	}
}

func TestPaperDetailMissingIs404(t *testing.T) {
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: t.TempDir()}))
	defer ts.Close()
	_, status, _ := getURL(t, ts.URL+"/papers/nope")
	if status != http.StatusNotFound {
		t.Fatalf("missing paper status = %d, want 404", status)
	}
}

func TestPaperPDFServedWhenPresent(t *testing.T) {
	dir := t.TempDir()
	writeParsedDoc(t, dir, "10-1000-ap", sampleParsedDoc)
	writeLocalPDF(t, dir, "10-1000-ap")
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()

	detailBody, _, _ := getURL(t, ts.URL+"/papers/10-1000-ap")
	if !strings.Contains(detailBody, "/papers/10-1000-ap/pdf") {
		t.Fatalf("paper detail missing PDF embed: %s", detailBody)
	}

	pdfBody, pdfStatus, pdfType := getURL(t, ts.URL+"/papers/10-1000-ap/pdf")
	if pdfStatus != http.StatusOK {
		t.Fatalf("GET pdf status = %d", pdfStatus)
	}
	if !strings.Contains(pdfType, "application/pdf") {
		t.Fatalf("pdf content-type = %q", pdfType)
	}
	if !strings.HasPrefix(pdfBody, "%PDF") {
		t.Fatalf("pdf body = %q", pdfBody)
	}
}

func TestPaperPDFAbsentIs404(t *testing.T) {
	dir := t.TempDir()
	writeParsedDoc(t, dir, "10-1000-ap", sampleParsedDoc)
	ts := httptest.NewServer(NewRouter(Config{ProjectPath: dir}))
	defer ts.Close()
	_, status, _ := getURL(t, ts.URL+"/papers/10-1000-ap/pdf")
	if status != http.StatusNotFound {
		t.Fatalf("absent pdf status = %d, want 404", status)
	}
}

func TestPaperIDRejectsTraversal(t *testing.T) {
	if _, ok := sanitizePaperID("../secret"); ok {
		t.Fatal("expected traversal id to be rejected")
	}
	if _, ok := sanitizePaperID("10-1000-ap"); !ok {
		t.Fatal("expected safe id to be accepted")
	}
}
