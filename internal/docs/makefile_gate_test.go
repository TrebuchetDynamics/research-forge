package docs

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/oss"
)

func TestMakeFmtCheck(t *testing.T) {
	makePath, err := exec.LookPath("make")
	if err != nil {
		t.Fatal(err)
	}
	gofmtPath, err := exec.LookPath("gofmt")
	if err != nil {
		t.Fatal(err)
	}
	makefile, err := filepath.Abs(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	// Do not inherit recursive-make flags or extra makefiles from the caller.
	for _, name := range []string{"MAKEFLAGS", "GNUMAKEFLAGS", "MFLAGS", "MAKELEVEL", "MAKEFILES"} {
		t.Setenv(name, "")
	}
	for _, tt := range []struct {
		name       string
		source     string
		stub       bool
		wantFail   bool
		diagnostic string
	}{
		{"formatter failure", "package fixture\n", true, true, "fixture formatter unavailable"},
		{"malformed", "package fixture\nfunc broken(\n", false, true, "expected"},
		{"unformatted", "package fixture\nvar answer=42\n", false, true, ""},
		{"formatted", "package fixture\n\nvar answer = 42\n", false, false, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			before := make(map[string][]byte)
			for _, tree := range []string{"cmd", "internal"} {
				if err := os.Mkdir(filepath.Join(dir, tree), 0o755); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(dir, tree, "fixture.go")
				if err := os.WriteFile(path, []byte(tt.source), 0o644); err != nil {
					t.Fatal(err)
				}
				before[path], err = os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
			}
			bin := filepath.Dir(gofmtPath)
			if tt.stub {
				bin = t.TempDir()
				if err := os.WriteFile(filepath.Join(bin, "gofmt"), []byte("#!/bin/sh\necho 'fixture formatter unavailable' >&2\nexit 7\n"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			cmd := exec.Command(makePath, "-f", makefile, "fmt-check")
			cmd.Dir = dir
			cmd.Env = oss.IsolatedGitEnv()
			output, runErr := cmd.CombinedOutput()
			if (runErr != nil) != tt.wantFail {
				t.Errorf("fmt-check error = %v, want failure %v:\n%s", runErr, tt.wantFail, output)
			}
			if !strings.Contains(string(output), tt.diagnostic) {
				t.Errorf("fmt-check missing diagnostic %q:\n%s", tt.diagnostic, output)
			}
			for path, original := range before {
				after, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(after, original) {
					t.Errorf("fmt-check rewrote %s", path)
				}
			}
		})
	}
}
