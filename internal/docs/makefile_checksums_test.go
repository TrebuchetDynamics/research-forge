package docs

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/oss"
)

func TestMakeChecksums(t *testing.T) {
	makePath, err := exec.LookPath("make")
	if err != nil {
		t.Fatal(err)
	}
	sha256sumPath, err := exec.LookPath("sha256sum")
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
		name  string
		mixed bool
		stale bool
		stub  bool
	}{
		{name: "binary absent"},
		{name: "binary stale", stale: true},
		{name: "mixed absent", mixed: true},
		{name: "mixed stale", mixed: true, stale: true},
		{name: "hasher failure", stub: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, "dist"), 0o755); err != nil {
				t.Fatal(err)
			}
			inputs := map[string][]byte{
				"dist/rforge-linux-amd64":       []byte("synthetic linux binary\x00\n"),
				"dist/rforge-windows-amd64.exe": []byte("synthetic windows binary\x00\n"),
			}
			if tt.mixed {
				inputs["dist/dependencies.json"] = []byte("{\"Path\":\"fixture\"}\n")
				inputs["dist/release notes.txt"] = []byte("synthetic release notes\n")
			}
			for path, content := range inputs {
				if err := os.WriteFile(filepath.Join(dir, path), content, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			manifestPath := filepath.Join(dir, "dist", "checksums.txt")
			if tt.stale {
				if err := os.WriteFile(manifestPath, []byte("stale manifest\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tt.stub {
				bin := t.TempDir()
				if err := os.WriteFile(filepath.Join(bin, "sha256sum"), []byte("#!/bin/sh\necho 'fixture hasher unavailable' >&2\nexit 7\n"), 0o755); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			}
			var first []byte
			for generation := 0; generation < 4; generation++ {
				if generation == 3 {
					path := "dist/rforge-linux-amd64"
					inputs[path] = []byte("mutated synthetic binary\n")
					if err := os.WriteFile(filepath.Join(dir, path), inputs[path], 0o644); err != nil {
						t.Fatal(err)
					}
				}
				cmd := exec.Command(makePath, "-f", makefile, "checksums")
				cmd.Dir = dir
				cmd.Env = oss.IsolatedGitEnv()
				output, runErr := cmd.CombinedOutput()
				if tt.stub {
					if runErr == nil || !strings.Contains(string(output), "fixture hasher unavailable") {
						t.Fatalf("checksums error = %v, want failure with hasher diagnostic:\n%s", runErr, output)
					}
					return
				}
				if runErr != nil {
					t.Fatalf("generation %d: %v:\n%s", generation, runErr, output)
				}
				manifest, err := os.ReadFile(manifestPath)
				if err != nil {
					t.Fatal(err)
				}
				if generation == 0 {
					first = manifest
				} else if bytes.Equal(manifest, first) != (generation < 3) {
					t.Errorf("generation %d: manifest equality does not match input changes", generation)
				}
				lines := strings.Split(strings.TrimSuffix(string(manifest), "\n"), "\n")
				if len(lines) != len(inputs) {
					t.Errorf("generation %d: want %d entries, got:\n%s", generation, len(inputs), manifest)
				}
				if strings.Contains(string(manifest), "dist/checksums.txt") {
					t.Errorf("generation %d: manifest includes itself:\n%s", generation, manifest)
				}
				for path, original := range inputs {
					entry := fmt.Sprintf("%x  %s", sha256.Sum256(original), path)
					count := 0
					for _, line := range lines {
						if line == entry {
							count++
						}
					}
					if count != 1 {
						t.Errorf("generation %d: entry %q occurs %d times", generation, entry, count)
					}
					after, err := os.ReadFile(filepath.Join(dir, path))
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(after, original) {
						t.Errorf("checksums rewrote %s", path)
					}
				}
				verify := exec.Command(sha256sumPath, "-c", "dist/checksums.txt")
				verify.Dir = dir
				verify.Env = oss.IsolatedGitEnv()
				if output, err := verify.CombinedOutput(); err != nil {
					t.Errorf("generation %d: checksum verification: %v:\n%s", generation, err, output)
				}
			}
		})
	}
}
