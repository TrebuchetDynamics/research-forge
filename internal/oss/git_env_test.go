package oss

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// A Git hook in a linked worktree exports GIT_DIR, GIT_WORK_TREE and
// GIT_INDEX_FILE. Git subprocesses must not inherit them, or they act on the
// caller's repository instead of their own working directory.
func TestGitCommandsIgnoreInheritedRepositoryEnvironment(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	decoy := filepath.Join(t.TempDir(), "decoy")
	if err := os.MkdirAll(decoy, 0o755); err != nil {
		t.Fatalf("mkdir decoy: %v", err)
	}
	runGit(t, decoy, "init", "-q")
	decoyConfig := filepath.Join(decoy, ".git", "config")
	before, err := os.ReadFile(decoyConfig)
	if err != nil {
		t.Fatalf("read decoy config: %v", err)
	}
	t.Setenv("GIT_DIR", filepath.Join(decoy, ".git"))
	t.Setenv("GIT_WORK_TREE", decoy)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(decoy, ".git", "index"))

	remote := filepath.Join(t.TempDir(), "remote")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatalf("mkdir remote: %v", err)
	}
	runGit(t, remote, "init", "-q")
	runGit(t, remote, "config", "user.email", "test@example.org")
	runGit(t, remote, "config", "user.name", "ResearchForge Test")
	if err := os.WriteFile(filepath.Join(remote, "README.md"), []byte("# fake repo\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGit(t, remote, "add", "README.md")
	runGit(t, remote, "commit", "-q", "-m", "initial")

	destination := filepath.Join(t.TempDir(), "clone")
	if err := (GitCloneRunner{}).Clone(context.Background(), CloneRequest{URL: remote, Destination: destination, Shallow: false}); err != nil {
		t.Fatalf("clone with inherited GIT_* environment: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "README.md")); err != nil {
		t.Fatalf("clone missing README: %v", err)
	}
	after, err := os.ReadFile(decoyConfig)
	if err != nil {
		t.Fatalf("read decoy config: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("decoy repository config changed:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if _, err := os.Stat(filepath.Join(decoy, "README.md")); err == nil {
		t.Fatalf("fixture file leaked into decoy work tree")
	}
}
