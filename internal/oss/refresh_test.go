package oss

import (
	"path/filepath"
	"testing"
)

func TestRegistryRefreshMetadataStoresScheduleStaleAndArchived(t *testing.T) {
	registry, err := OpenRegistry(filepath.Join(t.TempDir(), "oss.json"))
	if err != nil {
		t.Fatalf("OpenRegistry returned error: %v", err)
	}
	study, _ := NewRepositoryStudy(RepositoryStudyInput{Name: "owner/repo", Area: "area", RefreshInterval: "weekly"})
	if err := registry.Add(study); err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if err := registry.RefreshMetadata("owner/repo", RefreshMetadata{RefreshInterval: "daily", Stale: true, Archived: true}); err != nil {
		t.Fatalf("RefreshMetadata returned error: %v", err)
	}
	items, err := registry.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if items[0].RefreshInterval != "daily" || !items[0].Stale || !items[0].Archived {
		t.Fatalf("items = %#v", items)
	}
}
