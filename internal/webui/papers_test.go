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

func TestBuildPaperListRejectsSymlinkedParsedDocument(t *testing.T) {
	projectPath := t.TempDir()
	parsedDir := filepath.Join(projectPath, "parsed")
	if err := os.MkdirAll(parsedDir, 0o755); err != nil {
		t.Fatalf("mkdir project parsed: %v", err)
	}

	externalProject := t.TempDir()
	writeParsedDoc(t, externalProject, "external-private-paper", sampleParsedDoc)
	docPath := filepath.Join(parsedDir, "external-private-paper.json")
	if err := os.Symlink(filepath.Join(externalProject, "parsed", "external-private-paper.json"), docPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := BuildPaperList(projectPath); err == nil {
		t.Fatal("BuildPaperList accepted a symlinked parsed document")
	}
	if info, err := os.Lstat(docPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("parsed document symlink changed: info=%v err=%v", info, err)
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
	if !strings.Contains(listBody, `hx-trigger="refresh-papers from:body"`) {
		t.Fatalf("/papers missing project-switch refresh trigger: %s", listBody)
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

func TestPaperPDFSymlinkIsNotServed(t *testing.T) {
	projectPath := t.TempDir()
	writeParsedDoc(t, projectPath, "10-1000-ap", sampleParsedDoc)
	pdfDir := filepath.Join(projectPath, "documents", "open-access")
	if err := os.MkdirAll(pdfDir, 0o755); err != nil {
		t.Fatalf("mkdir project PDFs: %v", err)
	}
	externalPath := filepath.Join(t.TempDir(), "external-private.pdf")
	if err := os.WriteFile(externalPath, []byte("%PDF-1.4 external-private-content"), 0o644); err != nil {
		t.Fatalf("write external PDF: %v", err)
	}
	pdfPath := filepath.Join(pdfDir, "10-1000-ap.pdf")
	if err := os.Symlink(externalPath, pdfPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	ts := httptest.NewServer(NewRouter(Config{ProjectPath: projectPath}))
	defer ts.Close()
	body, status, _ := getURL(t, ts.URL+"/papers/10-1000-ap/pdf")
	if status != http.StatusNotFound {
		t.Fatalf("symlinked PDF status = %d, want 404; body=%q", status, body)
	}
	if strings.Contains(body, "external-private-content") {
		t.Fatalf("symlinked PDF disclosed external content: %q", body)
	}
	if info, err := os.Lstat(pdfPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("PDF symlink changed: info=%v err=%v", info, err)
	}
}

func TestPaperPDFAncestorSymlinkIsNotServed(t *testing.T) {
	projectPath := t.TempDir()
	writeParsedDoc(t, projectPath, "10-1000-ap", sampleParsedDoc)
	externalDocuments := filepath.Join(t.TempDir(), "documents")
	externalPDFDir := filepath.Join(externalDocuments, "open-access")
	if err := os.MkdirAll(externalPDFDir, 0o755); err != nil {
		t.Fatalf("mkdir external PDFs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(externalPDFDir, "10-1000-ap.pdf"), []byte("%PDF-1.4 external-ancestor-content"), 0o644); err != nil {
		t.Fatalf("write external PDF: %v", err)
	}
	documentsPath := filepath.Join(projectPath, "documents")
	if err := os.Symlink(externalDocuments, documentsPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	ts := httptest.NewServer(NewRouter(Config{ProjectPath: projectPath}))
	defer ts.Close()
	body, status, _ := getURL(t, ts.URL+"/papers/10-1000-ap/pdf")
	if status != http.StatusNotFound {
		t.Fatalf("ancestor-symlinked PDF status = %d, want 404; body=%q", status, body)
	}
	if strings.Contains(body, "external-ancestor-content") {
		t.Fatalf("ancestor-symlinked PDF disclosed external content: %q", body)
	}
	if info, err := os.Lstat(documentsPath); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("documents symlink changed: info=%v err=%v", info, err)
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
