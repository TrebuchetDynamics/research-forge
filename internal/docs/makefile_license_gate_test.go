package docs

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeLicenseApprovalGate(t *testing.T) {
	makePath, err := exec.LookPath("make")
	if err != nil {
		t.Fatal(err)
	}
	grepPath, err := exec.LookPath("grep")
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join("..", "..", "Makefile")
	makefile, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name     string
		stdout   string
		stderr   string
		exitCode int
		wantFail bool
	}{
		{"successful approved", `{"approved":true}`, "", 0, false},
		{"failed approved", `{"approved":true}`, "synthetic gh exit 7\n", 7, true},
		{"successful unapproved", `{"approved":false}`, "", 0, true},
		{"successful empty", "", "", 0, true},
		{"failed empty", "", "synthetic gh exit 7\n", 7, true},
		{"successful stderr approval", "", "{\"approved\":true}\n", 0, true},
		{"failed stderr approval", "", "{\"approved\":true}\nsynthetic gh exit 7\n", 7, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			if err := os.Mkdir(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			// Only these tools and the synthetic gh are reachable, including in recursive make.
			for name, path := range map[string]string{"make": makePath, "grep": grepPath} {
				if err := os.Symlink(path, filepath.Join(bin, name)); err != nil {
					t.Fatal(err)
				}
			}
			stub := []byte(fmt.Sprintf("#!/bin/sh\nprintf 'invoked\\n' > gh-invoked\nprintf '%%s' '%s'\nprintf '%%s' '%s' >&2\nexit %d\n", tt.stdout, tt.stderr, tt.exitCode))
			copyPath := filepath.Join(dir, "Makefile")
			stubPath := filepath.Join(bin, "gh")
			if err := os.WriteFile(copyPath, makefile, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(stubPath, stub, 0o755); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(makePath, "--no-print-directory", "license-decision-approval-gate")
			cmd.Dir = dir
			// An allowlist excludes Git redirection, make flags/overrides and user configuration.
			cmd.Env = []string{"PATH=" + bin, "HOME=" + dir, "LC_ALL=C"}
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			runErr := cmd.Run()
			if (runErr != nil) != tt.wantFail {
				t.Errorf("approval gate error = %v, want failure %v:\nstdout: %s\nstderr: %s", runErr, tt.wantFail, &stdout, &stderr)
			}
			if !strings.Contains(stderr.String(), tt.stderr) {
				t.Errorf("approval gate lost gh stderr %q:\n%s", tt.stderr, &stderr)
			}
			if tt.wantFail && !strings.Contains(stderr.String(), "license decision approval missing: issue #1 must report approved:true") {
				t.Errorf("approval gate missing rejection diagnostic:\n%s", &stderr)
			}
			invoked, err := os.ReadFile(filepath.Join(dir, "gh-invoked"))
			if err != nil || string(invoked) != "invoked\n" {
				t.Errorf("gh fixture was not invoked: %q, %v", invoked, err)
			}
			for path, original := range map[string][]byte{source: makefile, copyPath: makefile, stubPath: stub} {
				after, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(after, original) {
					t.Errorf("approval gate rewrote %s", path)
				}
			}
		})
	}
}
