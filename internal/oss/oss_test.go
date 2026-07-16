package oss

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestNewRepositoryStudyValidatesOwnerRepoAndStoresMetadata(t *testing.T) {
	study, err := NewRepositoryStudy(RepositoryStudyInput{
		Name:            "TrebuchetDynamics/research-forge",
		Area:            "literature tooling",
		RefreshInterval: "weekly",
	})
	if err != nil {
		t.Fatalf("NewRepositoryStudy returned error: %v", err)
	}
	if study.SchemaVersion != "1" || study.Name != "TrebuchetDynamics/research-forge" || study.Owner != "TrebuchetDynamics" || study.Repo != "research-forge" {
		t.Fatalf("study identity = %#v", study)
	}
	if study.ClonePath != filepath.Join("opensource", "clones", "TrebuchetDynamics", "research-forge") {
		t.Fatalf("ClonePath = %q", study.ClonePath)
	}
	if study.RefreshInterval != "weekly" || study.Area != "literature tooling" {
		t.Fatalf("study metadata = %#v", study)
	}
}

func TestNewRepositoryStudyRejectsUnsafeNames(t *testing.T) {
	badNames := []string{"research-forge", "../owner/repo", "owner/../repo", "owner/repo.git", "owner/repo/extra", "owner repo/name"}
	for _, name := range badNames {
		if _, err := NewRepositoryStudy(RepositoryStudyInput{Name: name}); err == nil {
			t.Fatalf("NewRepositoryStudy(%q) returned nil error", name)
		}
	}
}

func TestOpenRegistryDoesNotFollowDanglingSymlink(t *testing.T) {
	registryPath := filepath.Join(t.TempDir(), "oss.json")
	outsidePath := filepath.Join(t.TempDir(), "outside.json")
	if err := os.Symlink(outsidePath, registryPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	_, err := OpenRegistry(registryPath)
	if err == nil {
		t.Fatal("OpenRegistry succeeded with a dangling registry symlink")
	}
	if _, statErr := os.Stat(outsidePath); !os.IsNotExist(statErr) {
		t.Fatalf("outside path stat error = %v, want not exist", statErr)
	}
	info, lstatErr := os.Lstat(registryPath)
	if lstatErr != nil {
		t.Fatalf("lstat registry path: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("OpenRegistry replaced symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestRegistryAddDoesNotWriteThroughSymlinkedPath(t *testing.T) {
	registryPath := filepath.Join(t.TempDir(), "oss.json")
	registry, err := OpenRegistry(registryPath)
	if err != nil {
		t.Fatalf("OpenRegistry returned error: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.json")
	outsideBefore := []byte("[]\n")
	if err := os.WriteFile(outsidePath, outsideBefore, 0o640); err != nil {
		t.Fatalf("write outside registry: %v", err)
	}
	if err := os.Remove(registryPath); err != nil {
		t.Fatalf("remove registry path: %v", err)
	}
	if err := os.Symlink(outsidePath, registryPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	study, err := NewRepositoryStudy(RepositoryStudyInput{Name: "owner/repo"})
	if err != nil {
		t.Fatalf("NewRepositoryStudy returned error: %v", err)
	}

	if err := registry.Add(study); err == nil {
		t.Fatal("Add succeeded with a symlinked registry path")
	}
	outsideAfter, readErr := os.ReadFile(outsidePath)
	if readErr != nil {
		t.Fatalf("read outside registry: %v", readErr)
	}
	if !bytes.Equal(outsideAfter, outsideBefore) {
		t.Fatalf("Add wrote through registry symlink: got %q, want %q", outsideAfter, outsideBefore)
	}
	info, lstatErr := os.Lstat(registryPath)
	if lstatErr != nil {
		t.Fatalf("lstat registry path: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("Add replaced symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestRegistryListDoesNotReadThroughSymlinkedPath(t *testing.T) {
	registryPath := filepath.Join(t.TempDir(), "oss.json")
	registry, err := OpenRegistry(registryPath)
	if err != nil {
		t.Fatalf("OpenRegistry returned error: %v", err)
	}
	outsidePath := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outsidePath, []byte("[{\"Name\":\"outside/private\"}]\n"), 0o640); err != nil {
		t.Fatalf("write outside registry: %v", err)
	}
	if err := os.Remove(registryPath); err != nil {
		t.Fatalf("remove registry path: %v", err)
	}
	if err := os.Symlink(outsidePath, registryPath); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	if items, err := registry.List(); err == nil {
		t.Fatalf("List succeeded through a registry symlink: items=%#v", items)
	}
	info, lstatErr := os.Lstat(registryPath)
	if lstatErr != nil {
		t.Fatalf("lstat registry path: %v", lstatErr)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("List replaced registry symlink despite rejecting it: mode=%v", info.Mode())
	}
}

func TestRegistryAddPreservesPermissionsAndCleansStagingFiles(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "oss.json")
	if err := os.WriteFile(registryPath, []byte("[]\n"), 0o600); err != nil {
		t.Fatalf("write prior registry: %v", err)
	}
	registry, err := OpenRegistry(registryPath)
	if err != nil {
		t.Fatalf("OpenRegistry returned error: %v", err)
	}
	study, err := NewRepositoryStudy(RepositoryStudyInput{Name: "owner/repo"})
	if err != nil {
		t.Fatalf("NewRepositoryStudy returned error: %v", err)
	}
	if err := registry.Add(study); err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	info, err := os.Stat(registryPath)
	if err != nil {
		t.Fatalf("stat registry: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("registry mode = %o, want 600", info.Mode().Perm())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read registry directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(registryPath) {
		t.Fatalf("registry directory entries = %#v, want only %s", entries, filepath.Base(registryPath))
	}
}

func TestRegistryAddAndListRepositories(t *testing.T) {
	registry, err := OpenRegistry(filepath.Join(t.TempDir(), "oss.json"))
	if err != nil {
		t.Fatalf("OpenRegistry returned error: %v", err)
	}
	study, err := NewRepositoryStudy(RepositoryStudyInput{Name: "owner/repo", Area: "test area"})
	if err != nil {
		t.Fatalf("NewRepositoryStudy returned error: %v", err)
	}
	if err := registry.Add(study); err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	items, err := registry.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "owner/repo" || items[0].Area != "test area" {
		t.Fatalf("items = %#v", items)
	}
}

func TestResolveClonePathStaysInsideProjectOpenSourceClones(t *testing.T) {
	projectPath := t.TempDir()
	resolved, err := ResolveClonePath(projectPath, "owner/repo")
	if err != nil {
		t.Fatalf("ResolveClonePath returned error: %v", err)
	}
	want := filepath.Join(projectPath, "opensource", "clones", "owner", "repo")
	if resolved != want {
		t.Fatalf("resolved = %q, want %q", resolved, want)
	}
}

func TestLicenseFileDetection(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "LICENSE"), []byte("MIT License"), 0o644); err != nil {
		t.Fatalf("write license: %v", err)
	}
	license, err := DetectLicenseFile(dir)
	if err != nil {
		t.Fatalf("DetectLicenseFile returned error: %v", err)
	}
	if !license.Found || license.Path != filepath.Join(dir, "LICENSE") || license.Kind != "MIT" {
		t.Fatalf("license = %#v", license)
	}
}
