package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteForgePackageFixtureCompletesSpineToDone(t *testing.T) {
	projectPath := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := Execute([]string{"--json", "forge", "init", "--project", projectPath, "--question", "Do artificial photosynthesis catalysts improve solar fuel generation outcomes?"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	for _, step := range []struct{ cmd []string }{
		{[]string{"--json", "forge", "approve", "--project", projectPath, "--gate", "question approval", "--note", "accepted"}},
		{[]string{"--json", "forge", "approve", "--project", projectPath, "--gate", "protocol approval", "--note", "accepted"}},
		{[]string{"--json", "forge", "approve", "--project", projectPath, "--gate", "network/API approval", "--note", "accepted"}},
		{[]string{"--json", "forge", "source-fixture", "--project", projectPath}},
		{[]string{"--json", "forge", "reference-fixture", "--project", projectPath}},
		{[]string{"--json", "forge", "approve", "--project", projectPath, "--gate", "identity approval", "--note", "accepted"}},
		{[]string{"--json", "forge", "acquisition-fixture", "--project", projectPath}},
		{[]string{"--json", "forge", "approve", "--project", projectPath, "--gate", "parser arbitration approval", "--note", "accepted"}},
		{[]string{"--json", "forge", "next", "--project", projectPath}},
		{[]string{"--json", "forge", "approve", "--project", projectPath, "--gate", "screening approval", "--note", "accepted"}},
		{[]string{"--json", "forge", "approve", "--project", projectPath, "--gate", "evidence approval", "--note", "accepted"}},
		{[]string{"--json", "forge", "approve", "--project", projectPath, "--gate", "analysis approval", "--note", "accepted"}},
		{[]string{"--json", "forge", "approve", "--project", projectPath, "--gate", "claim approval", "--note", "accepted"}},
	} {
		stdout.Reset()
		stderr.Reset()
		if code := Execute(step.cmd, &stdout, &stderr); code != 0 {
			t.Fatalf("%v code=%d stderr=%s stdout=%s", step.cmd, code, stderr.String(), stdout.String())
		}
	}
	pkgDir := filepath.Join(t.TempDir(), "review.rforgepkg")
	stdout.Reset()
	stderr.Reset()
	code := Execute([]string{"--json", "forge", "package-fixture", "--project", projectPath, "--out", pkgDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("package-fixture code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"currentState":"done"`) || !strings.Contains(stdout.String(), `"ok":true`) {
		t.Fatalf("package-fixture stdout=%s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "package", "audit", pkgDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("audit package code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "reference-manager") || !strings.Contains(stdout.String(), "legal_acquisition") || !strings.Contains(stdout.String(), "document_assets") || !strings.Contains(stdout.String(), "accepted_evidence_support") {
		t.Fatalf("audit stdout=%s", stdout.String())
	}
}
