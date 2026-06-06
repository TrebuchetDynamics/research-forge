package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteProjectCreateWritesProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "created project") {
		t.Fatalf("stdout missing success message: %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "rforge.project.toml")); err != nil {
		t.Fatalf("manifest not created: %v", err)
	}
}

func TestExecuteVersion(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"version"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "rforge") {
		t.Fatalf("stdout missing version prefix: %q", stdout.String())
	}
}

func TestExecuteProjectInspect(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if code := Execute([]string{"project", "create", dir, "--title", "Demo Review"}, new(bytes.Buffer), new(bytes.Buffer)); code != 0 {
		t.Fatalf("project create exit code = %d", code)
	}
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)

	code := Execute([]string{"project", "inspect", dir}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Demo Review") {
		t.Fatalf("stdout missing project title: %q", stdout.String())
	}
}
