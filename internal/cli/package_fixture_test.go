package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutePackageFixtureCreatesOfflineReplayablePackage(t *testing.T) {
	out := filepath.Join(t.TempDir(), "review.rforgepkg")
	var stdout, stderr bytes.Buffer
	code := Execute([]string{"--json", "package", "fixture", "--out", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("package fixture code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "Artificial photosynthesis") && !strings.Contains(stdout.String(), "artificial photosynthesis") {
		t.Fatalf("fixture output missing topic: %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "package", "audit", out}, &stdout, &stderr); code != 0 {
		t.Fatalf("audit fixture code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"ok":true`) || !strings.Contains(stdout.String(), "accepted_evidence_support") {
		t.Fatalf("audit stdout=%s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Execute([]string{"--json", "package", "replay", out}, &stdout, &stderr); code != 0 {
		t.Fatalf("replay fixture code=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"ok":true`) {
		t.Fatalf("replay stdout=%s", stdout.String())
	}
}
