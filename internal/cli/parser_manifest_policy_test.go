package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

func TestExecuteParseWritesLicenseProvenanceManifest(t *testing.T) {
	project := t.TempDir()
	inputPath := filepath.Join(project, "s2orc.json")
	if err := os.WriteFile(inputPath, []byte(`{"title":"S2ORC Fixture","body_text":[{"section":"Intro","text":"Passage."}]}`), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "parse", "--paper", "paper-1", "--parser", "s2orc", "--s2orc", inputPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("parse code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	var env struct {
		Data struct {
			ManifestPath string `json:"manifestPath"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("decode stdout: %v\n%s", err, stdout.String())
	}
	var manifest parsing.ParserRunManifest
	data, err := os.ReadFile(env.Data.ManifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode manifest: %v\n%s", err, string(data))
	}
	if manifest.ParserSource == "" || manifest.LicenseConstraints == "" || manifest.Shareability == "" || manifest.OutputChecksum == "" || !manifest.ReviewerApprovalRequired {
		t.Fatalf("manifest missing policy fields: %#v", manifest)
	}
}

func TestExecuteParseManifestPoliciesJSONCoversExternalParsers(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", t.TempDir(), "parse", "manifest-policies"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("policies code=%d stderr=%s", code, stderr.String())
	}
	for _, want := range []string{"grobid", "s2orc", "papermage", "cermine", "science-parse", "anystyle"} {
		if !bytes.Contains(stdout.Bytes(), []byte(want)) {
			t.Fatalf("policies missing %s: %s", want, stdout.String())
		}
	}
}
