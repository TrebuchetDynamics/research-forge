package installer_test

import (
	"os"
	"path/filepath"
	"testing"
)

// An existing package must not let a successful list probe mask a failed install.
func TestInstallerQuietHostFailure(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		fail, dry  bool
	}{
		{name: "quiet-failure", mode: "--quiet", fail: true},
		{name: "verbose-failure", mode: "--verbose", fail: true},
		{name: "quiet-success", mode: "--quiet"},
		{name: "quiet-dry-run", mode: "--quiet", fail: true, dry: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newSandbox(t, "pi")
			s.seed("pi", "research-forge")
			legacy := s.path(".pi", "agent", "skills", "research-forge")
			if err := os.MkdirAll(legacy, 0o755); err != nil {
				t.Fatal(err)
			}
			s.writeScript(filepath.Join(legacy, "SKILL.md"), "previous working skill\n")
			if tc.fail {
				s.failOn("pi", "install")
			}
			beforeHome, beforeState := snapshot(t, s.home), snapshot(t, s.state)
			args := []string{"--only", "pi", tc.mode}
			if tc.dry {
				args = append(args, "--dry-run")
			}
			out, code, log := s.run(args...)
			if tc.fail && !tc.dry {
				if code != 1 {
					t.Errorf("exit = %d, want aggregate failure 1:\n%s", code, out)
				}
				wantContains(t, out, "pi: simulated failure for: install", "native failure")
				wantContains(t, out, "pi: failed", "failed component")
				wantContains(t, out, "0 installed, 0 skipped, 1 failed", "failure summary")
				wantNotContains(t, out, "Open a new host session", "no success advice")
				wantNotContains(t, out, "Next steps", "no success advice")
			} else if code != 0 {
				t.Errorf("exit = %d, want success:\n%s", code, out)
			} else if tc.dry {
				wantContains(t, out, "1 planned, 0 skipped, 0 failed", "dry-run summary")
			} else {
				wantContains(t, out, "1 installed, 0 skipped, 0 failed", "success summary")
				wantNotContains(t, out, "$ pi install", "quiet commands")
			}
			if tc.dry {
				wantContains(t, out, "$ pi install", "planned command")
				wantNotContains(t, log, "pi install", "dry-run must not install")
			}
			if tc.fail {
				if changed := diffSnapshots(beforeHome, snapshot(t, s.home)); len(changed) != 0 {
					t.Errorf("previous installation changed: %v", changed)
				}
				if changed := diffSnapshots(beforeState, snapshot(t, s.state)); len(changed) != 0 {
					t.Errorf("package state changed: %v", changed)
				}
			}
		})
	}
}

// A compatible old engine must not turn a failed checkout build into success.
func TestInstallerQuietSourceBuildFailure(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		fail, dry  bool
	}{
		{name: "quiet-failure", mode: "--quiet", fail: true},
		{name: "verbose-failure", mode: "--verbose", fail: true},
		{name: "quiet-success", mode: "--quiet"},
		{name: "quiet-dry-run", mode: "--quiet", fail: true, dry: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newSandbox(t)
			// Keep metadata probes offline and independent of inherited Git selectors.
			s.writeScript(filepath.Join(s.bin, "git"), "#!/usr/bin/env bash\nexit 1\n")
			s.writeScript(filepath.Join(s.bin, "go"), fakeGoScript)
			s.writeScript(filepath.Join(s.fixtures, "rforge-dev"), fakeRforgeScript)
			if tc.fail {
				s.writeScript(filepath.Join(s.bin, "go"), "#!/usr/bin/env bash\necho 'go: checkout build failed' >&2\nexit 37\n")
			}
			if err := os.MkdirAll(filepath.Join(s.repo, "cmd", "rforge"), 0o755); err != nil {
				t.Fatal(err)
			}
			binary := s.path(".local", "bin", "rforge")
			s.writeScript(binary, fakeRforgeScript+"# previous working engine\n")
			before := snapshot(t, s.home)
			args := []string{"--only", "binary", "--from-source", tc.mode}
			if tc.dry {
				args = append(args, "--dry-run")
			}
			out, code, log := s.run(args...)
			if tc.fail && !tc.dry {
				if code != 1 {
					t.Errorf("exit = %d, want aggregate failure 1:\n%s", code, out)
				}
				wantContains(t, out, "go: checkout build failed", "build failure")
				wantContains(t, out, "binary: failed", "failed component")
				wantContains(t, out, "0 installed, 0 skipped, 1 failed", "failure summary")
				wantNotContains(t, out, "Next steps", "no success advice")
			} else if code != 0 {
				t.Errorf("exit = %d, want success:\n%s", code, out)
			} else if tc.dry {
				wantContains(t, out, "1 planned, 0 skipped, 0 failed", "dry-run summary")
			} else {
				wantContains(t, out, "1 installed, 0 skipped, 0 failed", "success summary")
				wantNotContains(t, out, "go build", "quiet build output")
				data, err := os.ReadFile(binary)
				if err != nil || string(data) != fakeRforgeScript {
					t.Errorf("new engine was not installed: %q, %v", data, err)
				}
			}
			if tc.dry {
				wantContains(t, out, "go build", "planned build")
				wantNotContains(t, log, "go build", "dry-run must not build")
				wantNotContains(t, out, "go: checkout build failed", "dry-run must not execute Go")
			}
			if tc.fail {
				if changed := diffSnapshots(before, snapshot(t, s.home)); len(changed) != 0 {
					t.Errorf("previous engine changed: %v", changed)
				}
			}
		})
	}
}
