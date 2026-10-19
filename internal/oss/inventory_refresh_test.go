package oss

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRefreshInventoryGitHubMetadataUpdatesRepositoryEntries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/zotero/zotero" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"stargazers_count":123,"forks_count":45,"archived":false,"pushed_at":"2026-06-01T00:00:00Z","license":{"spdx_id":"AGPL-3.0-only"}}`))
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":"1","entries":[{"id":"zotero","name":"Zotero","repository":"zotero/zotero","area":"reference-management","disposition":"pattern-reference","licensePolicy":"study-only","note":"zotero.md","risk":"license review","nextSlice":"metadata refresh"},{"id":"metafor","name":"metafor","url":"https://www.metafor-project.org/","area":"meta-analysis","disposition":"adapter-only","licensePolicy":"adapter","note":"metafor.md","risk":"version capture","nextSlice":"sensitivity"}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	result, err := RefreshInventoryGitHubMetadata(path, GitHubMetadataOptions{BaseURL: server.URL, Client: server.Client()})
	if err != nil {
		t.Fatalf("RefreshInventoryGitHubMetadata returned error: %v", err)
	}
	if result.Refreshed != 1 || result.Skipped != 1 {
		t.Fatalf("result = %+v, want refreshed=1 skipped=1", result)
	}
	manifest, err := LoadInventoryManifest(path)
	if err != nil {
		t.Fatalf("reload manifest: %v", err)
	}
	entry := manifest.Entries[0]
	if entry.Stars != 123 || entry.Forks != 45 || entry.LicenseSPDX != "AGPL-3.0-only" || entry.Archived || entry.PushedAt != "2026-06-01T00:00:00Z" {
		t.Fatalf("entry metadata = %+v", entry)
	}
}
