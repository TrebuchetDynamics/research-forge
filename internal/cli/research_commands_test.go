package cli

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

func TestResearchScreenQueueCommandWritesCSVAndMarkdown(t *testing.T) {
	dir := t.TempDir()
	libraryPath := filepath.Join(dir, "library.json")
	writeJSONForCLITest(t, libraryPath, []library.PaperRecord{{
		Title:       "Cryptocurrency price prediction based on Xgboost, LightGBM and BNN",
		Abstract:    "Bitcoin cryptocurrency forecasting",
		Year:        2024,
		Identifiers: library.Identifiers{DOI: "10.example/lightgbm"},
		OpenAccess:  true,
	}})
	csvPath := filepath.Join(dir, "queue.csv")
	mdPath := filepath.Join(dir, "queue.md")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", dir, "research", "screen-queue", "--library", libraryPath, "--out", csvPath, "--markdown", mdPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(readFileForCLITest(t, csvPath), "direct_lightgbm_crypto") {
		t.Fatalf("csv missing screening group")
	}
	if !strings.Contains(readFileForCLITest(t, mdPath), "Structured screening queue") {
		t.Fatalf("markdown missing title")
	}
}

func TestResearchLeakageAuditCommandWritesJSONAndMarkdown(t *testing.T) {
	dir := t.TempDir()
	parsedDir := filepath.Join(dir, "parsed")
	if err := os.Mkdir(parsedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONForCLITest(t, filepath.Join(parsedDir, "paper.json"), parsing.ParsedDocument{
		SchemaVersion: "1",
		PaperID:       "paper-1",
		Title:         "Bitcoin order-flow validation",
		Sections:      []parsing.Section{{ID: "s1", Title: "Methods", Passages: []parsing.Passage{{ID: "p1", PaperID: "paper-1", SectionID: "s1", Text: "We forecast Bitcoin minute returns with order flow imbalance features. Training and testing use out-of-sample validation. Normalization is discussed."}}}},
	})
	jsonPath := filepath.Join(dir, "audit.json")
	mdPath := filepath.Join(dir, "audit.md")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", dir, "research", "leakage-audit", "--parsed", parsedDir, "--out", jsonPath, "--markdown", mdPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(readFileForCLITest(t, jsonPath), "global_preprocessing_possible") {
		t.Fatalf("audit json missing leakage flag")
	}
	if !strings.Contains(readFileForCLITest(t, mdPath), "Leakage-risk and feature-evidence triage") {
		t.Fatalf("audit markdown missing title")
	}
}

func writeJSONForCLITest(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFileForCLITest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
