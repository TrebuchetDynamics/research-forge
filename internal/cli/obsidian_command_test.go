package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── install ──────────────────────────────────────────────────────────────────

func stubObsidianLatestVersion(t *testing.T) {
	t.Helper()
	old := obsidianLatestVersionFunc
	obsidianLatestVersionFunc = func() string { return "1.8.10" }
	t.Cleanup(func() { obsidianLatestVersionFunc = old })
}

func TestObsidianInstallShowsDownloadURL(t *testing.T) {
	stubObsidianLatestVersion(t)
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	// Run with --dry-run so no actual download happens in tests
	code := Execute([]string{"obsidian", "install", "--dry-run"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr = %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "obsidianmd/obsidian-releases") {
		t.Errorf("install --dry-run should show download URL, got:\n%s", out)
	}
}

func TestObsidianInstallDryRunShowsDestination(t *testing.T) {
	stubObsidianLatestVersion(t)
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"obsidian", "install", "--dry-run"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr = %s", code, stderr.String())
	}
	// Should tell the user where it would install to
	out := stdout.String()
	if !strings.Contains(out, "obsidian") {
		t.Errorf("dry-run should mention install destination, got:\n%s", out)
	}
}

func TestObsidianInstallDryRunJSONOutput(t *testing.T) {
	stubObsidianLatestVersion(t)
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"--json", "obsidian", "install", "--dry-run"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("exit code = %d; stderr = %s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, `"ok":true`) {
		t.Errorf("JSON output missing ok:true:\n%s", out)
	}
	if !strings.Contains(out, "url") {
		t.Errorf("JSON output missing url field:\n%s", out)
	}
}

// ── open ─────────────────────────────────────────────────────────────────────

func TestObsidianOpenRequiresVaultFlag(t *testing.T) {
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"obsidian", "open"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2 for missing --vault, got %d", code)
	}
}

func TestObsidianOpenMissingVaultDirReturnsError(t *testing.T) {
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"obsidian", "open", "--vault", "/nonexistent/vault/path"}, stdout, stderr)
	if code == 0 {
		t.Error("expected non-zero exit for missing vault dir")
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "not found") && !strings.Contains(combined, "does not exist") && !strings.Contains(combined, "no such") {
		t.Errorf("error should mention vault not found, got:\n%s", combined)
	}
}

func TestObsidianOpenExistingVaultNoObsidianInstalled(t *testing.T) {
	vaultDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(vaultDir, "index.md"), []byte("# test vault\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := obsidianFindBinaryFunc
	obsidianFindBinaryFunc = func() (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { obsidianFindBinaryFunc = old })

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"obsidian", "open", "--vault", vaultDir}, stdout, stderr)

	if code == 0 {
		t.Fatal("expected missing Obsidian to fail without launching anything")
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "install") && !strings.Contains(combined, "not found") {
		t.Errorf("error for missing Obsidian should suggest install, got:\n%s", combined)
	}
}

func TestObsidianOpenDryRun(t *testing.T) {
	vaultDir := t.TempDir()
	os.WriteFile(filepath.Join(vaultDir, "index.md"), []byte("# vault\n"), 0o644)

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"obsidian", "open", "--vault", vaultDir, "--dry-run"}, stdout, stderr)
	if code != 0 {
		t.Fatalf("dry-run should succeed even without Obsidian installed; code=%d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, vaultDir) {
		t.Errorf("dry-run should echo vault path in output, got:\n%s", out)
	}
}

// ── dispatch ──────────────────────────────────────────────────────────────────

func TestObsidianUnknownSubcommand(t *testing.T) {
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"obsidian", "frobnicate"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2 for unknown subcommand, got %d", code)
	}
}

func TestObsidianNoSubcommand(t *testing.T) {
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"obsidian"}, stdout, stderr)
	if code != 2 {
		t.Errorf("expected exit 2 for no subcommand, got %d", code)
	}
}
