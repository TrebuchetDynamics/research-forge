package oss

import (
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
