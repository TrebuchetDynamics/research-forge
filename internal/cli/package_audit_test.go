package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutePackageAuditAndReplay(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Audit Package"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	writeJSONForCLITest(t, filepath.Join(project, "rforge.lock.json"), map[string]string{"version": "1"})
	writeAuditablePackageInputs(t, project)
	out := filepath.Join(t.TempDir(), "review.rforgepkg")
	if code := Execute([]string{"--project", project, "package", "create", "--out", out}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("package create code=%d", code)
	}
	var stdout, stderr bytes.Buffer
	if code := Execute([]string{"--json", "package", "audit", out}, &stdout, &stderr); code != 0 {
		t.Fatalf("audit code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"ok":true`) {
		t.Fatalf("audit stdout=%s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "package", "replay", out}, &stdout, &stderr); code != 0 {
		t.Fatalf("replay code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"ok":true`) {
		t.Fatalf("replay stdout=%s", stdout.String())
	}
}

func writeAuditablePackageInputs(t *testing.T, project string) {
	t.Helper()
	for _, dir := range []string{"analysis", "reports"} {
		if err := os.MkdirAll(filepath.Join(project, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	writeJSONForCLITest(t, filepath.Join(project, "analysis", "run.json"), map[string]any{"InputRows": []any{}})
	if err := os.WriteFile(filepath.Join(project, "reports", "report.md"), []byte("# Report\n"), 0o644); err != nil {
		t.Fatalf("write report: %v", err)
	}
	if err := os.WriteFile(filepath.Join(project, "data", "provenance.jsonl"), []byte("{\"action\":\"test\"}\n"), 0o644); err != nil {
		t.Fatalf("write provenance: %v", err)
	}
}

func TestExecutePackageVerificationJSONReturnsFailureExitCode(t *testing.T) {
	project := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", project, "--title", "Audit Failure"}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("project create code=%d", code)
	}
	writeJSONForCLITest(t, filepath.Join(project, "rforge.lock.json"), map[string]string{"version": "1"})
	writeAuditablePackageInputs(t, project)
	pkgDir := filepath.Join(t.TempDir(), "review.rforgepkg")
	if code := Execute([]string{"--project", project, "package", "create", "--out", pkgDir}, ioDiscard{}, ioDiscard{}); code != 0 {
		t.Fatalf("package create code=%d", code)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "project", "rforge.lock.json"), []byte("tampered\n"), 0o644); err != nil {
		t.Fatalf("tamper package: %v", err)
	}
	for _, command := range []string{"audit", "replay"} {
		t.Run(command, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			code := Execute([]string{"--json", "package", command, pkgDir}, &stdout, &stderr)
			if code != 1 {
				t.Fatalf("%s code=%d stdout=%s stderr=%s, want 1", command, code, stdout.String(), stderr.String())
			}
			var envelope struct {
				OK   bool `json:"ok"`
				Data struct {
					OK bool `json:"ok"`
				} `json:"data"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
				t.Fatalf("decode %s response: %v", command, err)
			}
			if envelope.OK || envelope.Data.OK {
				t.Fatalf("%s response=%s, want failed outer and data envelopes", command, stdout.String())
			}
		})
	}
}
