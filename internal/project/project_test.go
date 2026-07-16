package project

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/storage"
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
	if created.SchemaVersion != "1" {
		t.Fatalf("created schema version = %q, want 1", created.SchemaVersion)
	}
	if created.ManifestPath != filepath.Join(dir, "rforge.project.toml") {
		t.Fatalf("created manifest path = %q", created.ManifestPath)
	}
	if created.LockfilePath != filepath.Join(dir, "rforge.lock.json") {
		t.Fatalf("created lockfile path = %q", created.LockfilePath)
	}
	if created.ProvenancePath != filepath.Join(dir, "provenance", "events.jsonl") {
		t.Fatalf("created provenance path = %q", created.ProvenancePath)
	}
	if created.StoragePath != filepath.Join(dir, "data", "rforge.sqlite") {
		t.Fatalf("created storage path = %q", created.StoragePath)
	}
	if created.ArchiveMetadataPath != filepath.Join(dir, "rforge.archive.json") {
		t.Fatalf("created archive metadata path = %q", created.ArchiveMetadataPath)
	}

	archiveBytes, err := os.ReadFile(created.ArchiveMetadataPath)
	if err != nil {
		t.Fatalf("read archive metadata: %v", err)
	}
	var archive map[string]any
	if err := json.Unmarshal(archiveBytes, &archive); err != nil {
		t.Fatalf("archive metadata is not JSON: %v", err)
	}
	if archive["schemaVersion"] != "1" {
		t.Fatalf("archive schemaVersion = %#v, want 1", archive["schemaVersion"])
	}
	for _, key := range []string{"manifest", "lockfile", "provenance", "storage"} {
		value, ok := archive[key].(string)
		if !ok || value == "" {
			t.Fatalf("archive metadata missing relative %s path: %#v", key, archive)
		}
		if filepath.IsAbs(value) {
			t.Fatalf("archive metadata %s path is absolute: %q", key, value)
		}
	}

	manifestBytes, err := os.ReadFile(created.ManifestPath)
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

	if _, err := os.Stat(created.StoragePath); err != nil {
		t.Fatalf("sqlite database not initialized: %v", err)
	}

	lockBytes, err := os.ReadFile(created.LockfilePath)
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

	eventsBytes, err := os.ReadFile(created.ProvenancePath)
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

func TestCreateRejectsUnsafeProjectPaths(t *testing.T) {
	for _, path := range []string{"", "   ", "../escape", "safe/../escape"} {
		t.Run(path, func(t *testing.T) {
			if _, err := Create(path, CreateOptions{Title: "Unsafe"}); err == nil {
				t.Fatalf("Create(%q) returned nil error, want validation error", path)
			}
		})
	}
}

func TestCreateUsesInjectedEventID(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	_, err := Create(dir, CreateOptions{
		Title:   "Demo Review",
		EventID: func(time.Time) string { return "evt_test" },
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	events, err := ReadEvents(dir)
	if err != nil {
		t.Fatalf("ReadEvents returned error: %v", err)
	}
	if events[0].ID != "evt_test" {
		t.Fatalf("event ID = %q, want evt_test", events[0].ID)
	}
}

func TestCreateRollsBackWhenProvenanceAppendFails(t *testing.T) {
	invalidEvent := CreateOptions{
		Title:   "Demo Review",
		EventID: func(time.Time) string { return "" },
	}

	t.Run("removes a newly created workspace", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "demo")
		if _, err := Create(dir, invalidEvent); err == nil {
			t.Fatal("Create returned nil error, want provenance validation failure")
		}
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("failed creation left workspace %s: %v", dir, err)
		}
	})

	t.Run("removes only generated paths from an existing folder", func(t *testing.T) {
		dir := t.TempDir()
		assetPath := filepath.Join(dir, "notes.md")
		asset := []byte("existing research notes\n")
		if err := os.WriteFile(assetPath, asset, 0o640); err != nil {
			t.Fatalf("write existing asset: %v", err)
		}
		if _, err := Create(dir, invalidEvent); err == nil {
			t.Fatal("Create returned nil error, want provenance validation failure")
		}
		got, err := os.ReadFile(assetPath)
		if err != nil || !bytes.Equal(got, asset) {
			t.Fatalf("existing asset changed: data=%q err=%v", got, err)
		}
		for _, path := range []string{filepath.Join(dir, "data"), filepath.Join(dir, "provenance")} {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("failed creation left generated directory %s: %v", path, err)
			}
		}
	})

	t.Run("rejects a symlinked output before mutation", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(t.TempDir(), "outside.toml")
		original := []byte("outside content\n")
		if err := os.WriteFile(target, original, 0o640); err != nil {
			t.Fatalf("write symlink target: %v", err)
		}
		if err := os.Symlink(target, filepath.Join(dir, "rforge.project.toml")); err != nil {
			t.Fatalf("create manifest symlink: %v", err)
		}
		if _, err := Create(dir, CreateOptions{Title: "Demo Review"}); err == nil {
			t.Fatal("Create returned nil error for symlinked output")
		}
		got, err := os.ReadFile(target)
		if err != nil || !bytes.Equal(got, original) {
			t.Fatalf("symlink target changed: data=%q err=%v", got, err)
		}
		for _, path := range []string{filepath.Join(dir, "data"), filepath.Join(dir, "provenance")} {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("symlink preflight created directory %s: %v", path, err)
			}
		}
	})

	t.Run("does not rewrite a hard-linked existing output", func(t *testing.T) {
		dir := t.TempDir()
		outsidePath := filepath.Join(t.TempDir(), "outside.toml")
		original := []byte("outside manifest must not be rewritten\n")
		if err := os.WriteFile(outsidePath, original, 0o600); err != nil {
			t.Fatalf("write outside manifest: %v", err)
		}
		manifestPath := filepath.Join(dir, "rforge.project.toml")
		if err := os.Link(outsidePath, manifestPath); err != nil {
			t.Skipf("hard links are unavailable: %v", err)
		}
		fixedTime := time.Unix(1_600_000_000, 0)
		if err := os.Chtimes(outsidePath, fixedTime, fixedTime); err != nil {
			t.Fatalf("set outside manifest timestamps: %v", err)
		}
		before, err := os.Stat(outsidePath)
		if err != nil {
			t.Fatalf("stat outside manifest before Create: %v", err)
		}

		if _, err := Create(dir, invalidEvent); err == nil {
			t.Fatal("Create returned nil error, want provenance validation failure")
		}

		got, err := os.ReadFile(outsidePath)
		if err != nil {
			t.Fatalf("read outside manifest after rollback: %v", err)
		}
		if !bytes.Equal(got, original) {
			t.Fatalf("outside manifest changed: got %q, want %q", got, original)
		}
		after, err := os.Stat(outsidePath)
		if err != nil {
			t.Fatalf("stat outside manifest after rollback: %v", err)
		}
		if !after.ModTime().Equal(before.ModTime()) {
			t.Fatalf("outside manifest mtime changed: got %s, want %s", after.ModTime(), before.ModTime())
		}
	})

	t.Run("preserves an existing research folder", func(t *testing.T) {
		dir := t.TempDir()
		assetPath := filepath.Join(dir, "paper.pdf")
		asset := []byte("%PDF-1.4 existing research asset")
		if err := os.WriteFile(assetPath, asset, 0o640); err != nil {
			t.Fatalf("write existing asset: %v", err)
		}
		storagePath := filepath.Join(dir, "data", "rforge.sqlite")
		if err := os.MkdirAll(filepath.Dir(storagePath), 0o755); err != nil {
			t.Fatalf("create existing data directory: %v", err)
		}
		store, err := storage.Initialize(storagePath)
		if err != nil {
			t.Fatalf("initialize existing storage: %v", err)
		}
		if err := store.Close(); err != nil {
			t.Fatalf("close existing storage: %v", err)
		}
		storageBytes, err := os.ReadFile(storagePath)
		if err != nil {
			t.Fatalf("read existing storage: %v", err)
		}
		prior := map[string][]byte{
			filepath.Join(dir, "rforge.project.toml"):        []byte("existing manifest\n"),
			filepath.Join(dir, "rforge.archive.json"):        []byte("existing archive metadata\n"),
			filepath.Join(dir, "provenance", "events.jsonl"): []byte("existing provenance\n"),
			storagePath:                        storageBytes,
			storagePath + ".pre-migration.bak": []byte("existing migration backup\n"),
		}
		for path, data := range prior {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("create existing path parent: %v", err)
			}
			if err := os.WriteFile(path, data, 0o640); err != nil {
				t.Fatalf("write existing project path %s: %v", path, err)
			}
		}

		if _, err := Create(dir, invalidEvent); err == nil {
			t.Fatal("Create returned nil error, want provenance validation failure")
		}
		for path, want := range prior {
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read restored path %s: %v", path, err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("restored %s = %q, want %q", path, got, want)
			}
		}
		gotAsset, err := os.ReadFile(assetPath)
		if err != nil || !bytes.Equal(gotAsset, asset) {
			t.Errorf("existing asset changed: data=%q err=%v", gotAsset, err)
		}
		for _, path := range []string{
			filepath.Join(dir, "rforge.lock.json"),
			storagePath + "-journal",
			storagePath + "-wal",
			storagePath + "-shm",
		} {
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Errorf("failed creation left generated path %s: %v", path, err)
			}
		}
	})
}

func TestReadEventsReturnsProjectCreateEvent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	_, err := Create(dir, CreateOptions{Title: "Demo Review"})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	events, err := ReadEvents(dir)
	if err != nil {
		t.Fatalf("ReadEvents returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if events[0].Action != "project.create" {
		t.Fatalf("Action = %q, want project.create", events[0].Action)
	}
	if events[0].Target != dir {
		t.Fatalf("Target = %q, want %q", events[0].Target, dir)
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
