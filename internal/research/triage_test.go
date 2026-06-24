package research

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestParsedTextDocumentChunksPassages(t *testing.T) {
	doc := ParsedTextDocument("paper-1", "A Paper", "First paragraph about Bitcoin order flow and validation.\n\nSecond paragraph about LightGBM features and leakage controls.", 60)
	if doc.ParserName != "pdftotext-sectionizer" || doc.PaperID != "paper-1" {
		t.Fatalf("unexpected parsed document metadata: %#v", doc)
	}
	if len(doc.Sections) != 1 || len(doc.Sections[0].Passages) != 2 {
		t.Fatalf("expected 2 passages, got %#v", doc.Sections)
	}
}

func TestBuildScreeningQueueRanksRelevantCryptoPapers(t *testing.T) {
	dir := t.TempDir()
	libraryPath := filepath.Join(dir, "library.json")
	papers := []library.PaperRecord{
		{Title: "Cryptocurrency price prediction based on Xgboost, LightGBM and BNN", Abstract: "Bitcoin cryptocurrency forecasting", Year: 2024, Identifiers: library.Identifiers{DOI: "10.example/lightgbm"}, OpenAccess: true},
		{Title: "Diagnosis of diabetes using LightGBM", Abstract: "health study", Year: 2021, Identifiers: library.Identifiers{DOI: "10.example/diabetes"}},
	}
	writeJSONForTest(t, libraryPath, papers)
	queue, err := BuildScreeningQueue(libraryPath, "")
	if err != nil {
		t.Fatalf("BuildScreeningQueue() error = %v", err)
	}
	if len(queue) != 2 {
		t.Fatalf("queue length = %d", len(queue))
	}
	if queue[0].DOI != "10.example/lightgbm" || queue[0].Decision != "include-review" {
		t.Fatalf("top queue row = %#v", queue[0])
	}
	if queue[1].Decision == "include-review" {
		t.Fatalf("irrelevant health paper was included: %#v", queue[1])
	}
	var csvOut bytes.Buffer
	if err := WriteScreeningCSV(&csvOut, queue); err != nil {
		t.Fatalf("WriteScreeningCSV() error = %v", err)
	}
	if !strings.Contains(csvOut.String(), "direct_lightgbm_crypto") {
		t.Fatalf("csv missing matched group: %s", csvOut.String())
	}
}

func TestBuildLeakageAuditFromTextDirExtractsEvidence(t *testing.T) {
	dir := t.TempDir()
	textDir := filepath.Join(dir, "extracted-text")
	if err := os.Mkdir(textDir, 0o755); err != nil {
		t.Fatal(err)
	}
	text := "Bitcoin order flow prediction\n\nWe forecast Bitcoin minute returns with order book imbalance and spread features. Training and testing are chronological, but normalization and random forest baselines are discussed."
	if err := os.WriteFile(filepath.Join(textDir, "paper-1.txt"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := BuildLeakageAuditFromTextDir(textDir, 80)
	if err != nil {
		t.Fatalf("BuildLeakageAuditFromTextDir() error = %v", err)
	}
	if len(rows) != 1 || rows[0].PaperID != "paper-1" || rows[0].Title != "Bitcoin order flow prediction" {
		t.Fatalf("unexpected rows: %#v", rows)
	}
	if len(rows[0].FeatureEvidence) == 0 || !rows[0].LeakageFlags["global_preprocessing_possible"] {
		t.Fatalf("missing text evidence: %#v", rows[0])
	}
}

func TestBuildLeakageAuditExtractsEvidence(t *testing.T) {
	dir := t.TempDir()
	parsedDir := filepath.Join(dir, "parsed")
	if err := os.Mkdir(parsedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	doc := parsing.ParsedDocument{
		SchemaVersion: "1",
		PaperID:       "paper-1",
		Title:         "Bitcoin order flow prediction",
		Sections:      []parsing.Section{{ID: "s1", Title: "Methods", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "We forecast Bitcoin minute returns with order book imbalance and spread features. Training and testing are chronological, but normalization and random forest baselines are discussed."}}}},
	}
	writeJSONForTest(t, filepath.Join(parsedDir, "paper-1.json"), doc)
	rows, err := BuildLeakageAudit(parsedDir)
	if err != nil {
		t.Fatalf("BuildLeakageAudit() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows length = %d", len(rows))
	}
	row := rows[0]
	if len(row.ValidationEvidence) == 0 || len(row.FeatureEvidence) == 0 || len(row.LeakageRiskEvidence) == 0 {
		t.Fatalf("missing evidence: %#v", row)
	}
	if !row.LeakageFlags["global_preprocessing_possible"] {
		t.Fatalf("expected normalization leakage flag: %#v", row.LeakageFlags)
	}
	markdown := LeakageAuditMarkdown(rows)
	if !strings.Contains(markdown, "Bitcoin order flow prediction") || !strings.Contains(markdown, "Leakage flags") {
		t.Fatalf("unexpected markdown: %s", markdown)
	}
}

func writeJSONForTest(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
