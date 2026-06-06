package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateWritesManifestLockfileAndProvenance(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	clock := func() time.Time { return time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC) }

	created, err := Create(dir, CreateOptions{Title: "Demo Review", Clock: clock})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.Path != dir {
		t.Fatalf("created path = %q, want %q", created.Path, dir)
	}

	manifestBytes, err := os.ReadFile(filepath.Join(dir, "rforge.project.toml"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest := string(manifestBytes)
	for _, want := range []string{
		`schema_version = "1"`,
		`title = "Demo Review"`,
		`storage_mode = "sqlite"`,
	} {
		if !strings.Contains(manifest, want) {
			t.Fatalf("manifest missing %q:\n%s", want, manifest)
		}
	}

	lockBytes, err := os.ReadFile(filepath.Join(dir, "rforge.lock.json"))
	if err != nil {
		t.Fatalf("read lockfile: %v", err)
	}
	var lock map[string]any
	if err := json.Unmarshal(lockBytes, &lock); err != nil {
		t.Fatalf("lockfile is not JSON: %v", err)
	}
	if lock["schemaVersion"] != "1" {
		t.Fatalf("lock schemaVersion = %#v, want 1", lock["schemaVersion"])
	}

	eventsBytes, err := os.ReadFile(filepath.Join(dir, "provenance", "events.jsonl"))
	if err != nil {
		t.Fatalf("read provenance events: %v", err)
	}
	line := strings.TrimSpace(string(eventsBytes))
	var event map[string]any
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		t.Fatalf("event is not JSON: %v\n%s", err, line)
	}
	if event["id"] == "" {
		t.Fatalf("event id is required: %#v", event)
	}
	if event["actor"] != "rforge" {
		t.Fatalf("event actor = %#v, want rforge", event["actor"])
	}
	if event["action"] != "project.create" {
		t.Fatalf("event action = %#v, want project.create", event["action"])
	}
	if event["target"] != dir {
		t.Fatalf("event target = %#v, want %q", event["target"], dir)
	}
	if event["outputs"] == nil {
		t.Fatalf("event outputs are required: %#v", event)
	}
	if event["warnings"] == nil {
		t.Fatalf("event warnings are required: %#v", event)
	}
}

func TestInspectReadsCreatedProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	_, err := Create(dir, CreateOptions{Title: "Demo Review"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	inspected, err := Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if inspected.Path != dir {
		t.Fatalf("Path = %q, want %q", inspected.Path, dir)
	}
	if inspected.Title != "Demo Review" {
		t.Fatalf("Title = %q, want Demo Review", inspected.Title)
	}
	if inspected.StorageMode != "sqlite" {
		t.Fatalf("StorageMode = %q, want sqlite", inspected.StorageMode)
	}
}

func TestListReadsProjectsUnderRoot(t *testing.T) {
	root := t.TempDir()
	_, err := Create(filepath.Join(root, "b"), CreateOptions{Title: "Beta"})
	if err != nil {
		t.Fatalf("Create beta: %v", err)
	}
	_, err = Create(filepath.Join(root, "a"), CreateOptions{Title: "Alpha"})
	if err != nil {
		t.Fatalf("Create alpha: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "not-a-project.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("write non-project: %v", err)
	}

	projects, err := List(root)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("len(projects) = %d, want 2: %#v", len(projects), projects)
	}
	if projects[0].Title != "Alpha" || projects[1].Title != "Beta" {
		t.Fatalf("projects not sorted by path/title as expected: %#v", projects)
	}
}
