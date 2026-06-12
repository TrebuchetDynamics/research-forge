package oss

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

type fakeCloneRunner struct {
	calls []CloneRequest
}

func (f *fakeCloneRunner) Clone(ctx context.Context, request CloneRequest) error {
	f.calls = append(f.calls, request)
	return nil
}

func TestCloneRepositoryUsesShallowCloneRunnerAndSafePath(t *testing.T) {
	projectPath := t.TempDir()
	runner := &fakeCloneRunner{}
	result, err := CloneRepository(context.Background(), projectPath, "owner/repo", "https://example.org/owner/repo.git", runner)
	if err != nil {
		t.Fatalf("CloneRepository returned error: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("calls = %#v", runner.calls)
	}
	call := runner.calls[0]
	if !call.Shallow || call.Depth != 1 || call.URL != "https://example.org/owner/repo.git" || call.Destination != result.Path {
		t.Fatalf("call = %#v result = %#v", call, result)
	}
	if !reflect.DeepEqual(call.Args(), []string{"clone", "--depth", "1", "https://example.org/owner/repo.git", result.Path}) {
		t.Fatalf("args = %#v", call.Args())
	}
}

func TestGitCloneRunnerClonesLocalFakeRepositoryShallow(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	remote := filepath.Join(t.TempDir(), "remote")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatalf("mkdir remote: %v", err)
	}
	runGit(t, remote, "init")
	runGit(t, remote, "config", "user.email", "test@example.org")
	runGit(t, remote, "config", "user.name", "ResearchForge Test")
	if err := os.WriteFile(filepath.Join(remote, "README.md"), []byte("# fake repo\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	runGit(t, remote, "add", "README.md")
	runGit(t, remote, "commit", "-m", "initial")

	projectPath := t.TempDir()
	result, err := CloneRepository(context.Background(), projectPath, "owner/repo", remote, GitCloneRunner{})
	if err != nil {
		t.Fatalf("CloneRepository returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(result.Path, "README.md")); err != nil {
		t.Fatalf("missing cloned README: %v", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
