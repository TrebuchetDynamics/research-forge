package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProvenanceValidateAcceptsConformingFile verifies the validator passes a
// provenance.json that follows the versioned schema: enum depth, string-only
// errors, and all required fields present.
func TestProvenanceValidateAcceptsConformingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provenance.json")
	writeProvenanceFixture(t, path, map[string]any{
		"schema_version": "1",
		"question":       "test question",
		"depth":          "comprehensive",
		"sources":        []string{"openalex", "arxiv"},
		"queries":        []string{"q1"},
		"timestamp":      "2026-07-14T00:00:00Z",
		"rforge_version": provenanceVersionFixture(),
		"outputs":        []string{"report.md", "provenance.json"},
		"errors":         []string{},
	})
	stdout := new(bytes.Buffer)
	code := Execute([]string{"provenance", "validate", path}, stdout, new(bytes.Buffer))
	if code != 0 {
		t.Fatalf("exit code = %d, stdout = %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String(), "unverified") || !strings.Contains(stdout.String(), "warning") {
		t.Fatalf("legacy provenance must be downgraded with a warning: %s", stdout)
	}
}

// TestProvenanceValidateRejectsFreeFormDepth verifies the validator rejects a
// depth value outside the quick|standard|comprehensive enum, which is the
// schema-drift pattern observed across real provenance ("standard-plus",
// "comprehensive-lite", "standard search sweep script").
func TestProvenanceValidateRejectsFreeFormDepth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provenance.json")
	writeProvenanceFixture(t, path, map[string]any{
		"schema_version": "1",
		"question":       "test question",
		"depth":          "standard-plus",
		"sources":        []string{"openalex"},
		"queries":        []string{"q1"},
		"timestamp":      "2026-07-14T00:00:00Z",
		"rforge_version": provenanceVersionFixture(),
		"outputs":        []string{"report.md"},
		"errors":         []string{},
	})
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	code := Execute([]string{"--json", "provenance", "validate", path}, stdout, stderr)
	if code == 0 {
		t.Fatalf("validator must reject free-form depth 'standard-plus'; stdout = %s", stdout.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout.String())
	}
	errBody := envelope["error"].(map[string]any)
	if !strings.Contains(errBody["message"].(string), "depth") {
		t.Fatalf("error must name the depth field; got %v", errBody["message"])
	}
}

// TestProvenanceValidateRejectsNonStringErrors verifies the validator rejects
// nested JSON objects in the errors array, which break jq aggregation across
// the provenance corpus.
func TestProvenanceValidateRejectsNonStringErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provenance.json")
	writeProvenanceFixture(t, path, map[string]any{
		"schema_version": "1",
		"question":       "test question",
		"depth":          "standard",
		"sources":        []string{"openalex"},
		"queries":        []string{"q1"},
		"timestamp":      "2026-07-14T00:00:00Z",
		"rforge_version": provenanceVersionFixture(),
		"outputs":        []string{"report.md"},
		"errors":         []any{map[string]any{"repo": "example/repo", "error": "404"}},
	})
	stdout := new(bytes.Buffer)
	code := Execute([]string{"--json", "provenance", "validate", path}, stdout, new(bytes.Buffer))
	if code == 0 {
		t.Fatalf("validator must reject non-string errors entries; stdout = %s", stdout.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout.String())
	}
	errBody := envelope["error"].(map[string]any)
	if !strings.Contains(errBody["message"].(string), "errors") {
		t.Fatalf("error must name the errors field; got %v", errBody["message"])
	}
}

// TestProvenanceValidateRejectsMissingVersion verifies the validator rejects a
// missing or empty rforge_version, the field most often set to "none" or "dev"
// across real provenance.
func TestProvenanceValidateRejectsMissingVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provenance.json")
	writeProvenanceFixture(t, path, map[string]any{
		"schema_version": "1",
		"question":       "test question",
		"depth":          "standard",
		"sources":        []string{"openalex"},
		"queries":        []string{"q1"},
		"timestamp":      "2026-07-14T00:00:00Z",
		"outputs":        []string{"report.md"},
		"errors":         []string{},
	})
	stdout := new(bytes.Buffer)
	code := Execute([]string{"--json", "provenance", "validate", path}, stdout, new(bytes.Buffer))
	if code == 0 {
		t.Fatalf("validator must reject missing rforge_version; stdout = %s", stdout.String())
	}
	var envelope map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, stdout.String())
	}
	errBody := envelope["error"].(map[string]any)
	if !strings.Contains(errBody["message"].(string), "rforge_version") {
		t.Fatalf("error must name the rforge_version field; got %v", errBody["message"])
	}
}

func TestProvenanceValidateRejectsUnsupportedSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provenance.json")
	writeProvenanceFixture(t, path, map[string]any{
		"schema_version": "3",
		"question":       "test question",
		"depth":          "standard",
		"sources":        []string{"openalex"},
		"timestamp":      "2026-07-14T00:00:00Z",
		"rforge_version": provenanceVersionFixture(),
		"outputs":        []string{"report.md"},
		"errors":         []string{},
	})
	stdout := new(bytes.Buffer)
	code := Execute([]string{"--json", "provenance", "validate", path}, stdout, new(bytes.Buffer))
	if code == 0 {
		t.Fatalf("validator must reject unsupported schema_version; stdout = %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "schema_version") {
		t.Fatalf("error must name schema_version; stdout = %s", stdout.String())
	}
}

func TestProvenanceValidateRejectsFreeFormVersionString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "provenance.json")
	writeProvenanceFixture(t, path, map[string]any{
		"schema_version": "1",
		"question":       "test question",
		"depth":          "standard",
		"sources":        []string{"openalex"},
		"timestamp":      "2026-07-14T00:00:00Z",
		"rforge_version": "rforge v0.1.17 (fe2f413, 2026-07-08T01:25:35Z)",
		"outputs":        []string{"report.md"},
		"errors":         []string{},
	})
	stdout := new(bytes.Buffer)
	code := Execute([]string{"--json", "provenance", "validate", path}, stdout, new(bytes.Buffer))
	if code == 0 {
		t.Fatalf("validator must reject free-form rforge_version strings; stdout = %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "rforge_version") {
		t.Fatalf("error must name rforge_version; stdout = %s", stdout.String())
	}
}

// assertEngineReceipt checks command output through the public validation CLI.
func assertEngineReceipt(t *testing.T, dir string, output string) {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(dir, "provenance", "runs", "*.json"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("expected one immutable run receipt, got %v (%v)", paths, err)
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	var record struct {
		Outputs []struct {
			Path string `json:"path"`
		} `json:"outputs"`
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range record.Outputs {
		if entry.Path == output {
			found = true
		}
	}
	if !found {
		t.Fatalf("receipt does not cover %s: %s", output, data)
	}
	var stdout, stderr bytes.Buffer
	if code := Execute([]string{"--json", "provenance", "validate", paths[0]}, &stdout, &stderr); code != 0 {
		t.Fatalf("receipt validation: %s %s", &stdout, &stderr)
	}
	if !strings.Contains(stdout.String(), `"provenance_level":"engine_verified"`) {
		t.Fatalf("not engine verified: %s", &stdout)
	}
}

func provenanceVersionFixture() map[string]any {
	return map[string]any{"version": "v0.1.17", "commit": "fe2f413", "date": "2026-07-08T01:25:35Z"}
}

func writeProvenanceFixture(t *testing.T, path string, data map[string]any) {
	t.Helper()
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
