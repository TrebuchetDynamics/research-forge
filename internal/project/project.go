package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/storage"
)

const schemaVersion = "1"

// Project is a local ResearchForge workspace.
type Project struct {
	Path                string
	Title               string
	StorageMode         string
	SchemaVersion       string
	ManifestPath        string
	LockfilePath        string
	ProvenancePath      string
	StoragePath         string
	ArchiveMetadataPath string
}

// Asset is a pre-existing local research asset discovered before import.
type Asset struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Imported bool   `json:"imported"`
}

// Event is a provenance event recorded in a ResearchForge project.
type Event = provenance.Event

// CreateOptions configures project creation.
type CreateOptions struct {
	Title   string
	Clock   func() time.Time
	EventID func(time.Time) string
}

type creationFileSnapshot struct {
	path    string
	data    []byte
	mode    os.FileMode
	existed bool
}

type creationTransaction struct {
	files   []creationFileSnapshot
	newDirs []string
}

func beginCreationTransaction(dirs, files []string) (creationTransaction, error) {
	tx := creationTransaction{files: make([]creationFileSnapshot, 0, len(files)), newDirs: make([]string, 0, len(dirs))}
	for _, dir := range dirs {
		info, err := os.Lstat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				tx.newDirs = append(tx.newDirs, dir)
				continue
			}
			return creationTransaction{}, err
		}
		if !info.IsDir() {
			return creationTransaction{}, fmt.Errorf("project directory path is not a directory: %s", dir)
		}
	}
	for _, path := range files {
		info, err := os.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				tx.files = append(tx.files, creationFileSnapshot{path: path})
				continue
			}
			return creationTransaction{}, err
		}
		if !info.Mode().IsRegular() {
			return creationTransaction{}, fmt.Errorf("project output path is not a regular file: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return creationTransaction{}, err
		}
		tx.files = append(tx.files, creationFileSnapshot{path: path, data: data, mode: info.Mode(), existed: true})
	}
	return tx, nil
}

func (tx creationTransaction) rollback() error {
	failures := make([]string, 0)
	for i := len(tx.files) - 1; i >= 0; i-- {
		snapshot := tx.files[i]
		var err error
		if snapshot.existed {
			err = filetxn.Replace(snapshot.path, snapshot.data, snapshot.mode)
		} else {
			err = os.Remove(snapshot.path)
			if os.IsNotExist(err) {
				err = nil
			}
		}
		if err != nil {
			failures = append(failures, fmt.Sprintf("restore %s: %v", snapshot.path, err))
		}
	}
	for i := len(tx.newDirs) - 1; i >= 0; i-- {
		if err := os.Remove(tx.newDirs[i]); err != nil && !os.IsNotExist(err) {
			failures = append(failures, fmt.Sprintf("remove created directory %s: %v", tx.newDirs[i], err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("roll back project creation: %s", strings.Join(failures, "; "))
	}
	return nil
}

func replaceProjectFile(path string, data []byte, defaultMode os.FileMode) error {
	info, err := os.Lstat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("project file path is not a regular file: %s", path)
		}
		defaultMode = info.Mode()
	} else if !os.IsNotExist(err) {
		return err
	}
	return filetxn.Replace(path, data, defaultMode)
}

// Create initializes a ResearchForge project workspace.
func Create(path string, opts CreateOptions) (Project, error) {
	if err := ValidatePath(path); err != nil {
		return Project{}, err
	}
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		return Project{}, fmt.Errorf("project title is required")
	}
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}

	manifestPath := filepath.Join(path, "rforge.project.toml")
	lockfilePath := filepath.Join(path, "rforge.lock.json")
	provenancePath := filepath.Join(path, "provenance", "events.jsonl")
	storagePath := filepath.Join(path, "data", "rforge.sqlite")
	archiveMetadataPath := filepath.Join(path, "rforge.archive.json")
	tx, err := beginCreationTransaction(
		[]string{path, filepath.Join(path, "provenance"), filepath.Join(path, "data")},
		[]string{
			storagePath,
			storagePath + ".pre-migration.bak",
			storagePath + "-journal",
			storagePath + "-wal",
			storagePath + "-shm",
			manifestPath,
			lockfilePath,
			archiveMetadataPath,
			provenancePath,
		},
	)
	if err != nil {
		return Project{}, err
	}
	fail := func(cause error) (Project, error) {
		if rollbackErr := tx.rollback(); rollbackErr != nil {
			return Project{}, fmt.Errorf("%v; %w", cause, rollbackErr)
		}
		return Project{}, cause
	}

	if err := os.MkdirAll(filepath.Join(path, "provenance"), 0o755); err != nil {
		return fail(err)
	}
	if err := os.MkdirAll(filepath.Join(path, "data"), 0o755); err != nil {
		return fail(err)
	}

	store, err := storage.Initialize(storagePath)
	if err != nil {
		return fail(err)
	}
	if err := store.Close(); err != nil {
		return fail(err)
	}

	manifest := fmt.Sprintf("schema_version = %q\ntitle = %q\nstorage_mode = %q\n", schemaVersion, title, "sqlite")
	if err := replaceProjectFile(manifestPath, []byte(manifest), 0o644); err != nil {
		return fail(err)
	}

	lock := map[string]any{
		"schemaVersion": schemaVersion,
		"createdAt":     clock().UTC().Format(time.RFC3339),
		"tools":         map[string]any{},
	}
	lockBytes, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fail(err)
	}
	lockBytes = append(lockBytes, '\n')
	if err := replaceProjectFile(lockfilePath, lockBytes, 0o644); err != nil {
		return fail(err)
	}

	archiveMetadata := map[string]any{
		"schemaVersion": schemaVersion,
		"title":         title,
		"storageMode":   "sqlite",
		"manifest":      "rforge.project.toml",
		"lockfile":      "rforge.lock.json",
		"provenance":    "provenance/events.jsonl",
		"storage":       "data/rforge.sqlite",
	}
	archiveBytes, err := json.MarshalIndent(archiveMetadata, "", "  ")
	if err != nil {
		return fail(err)
	}
	archiveBytes = append(archiveBytes, '\n')
	if err := replaceProjectFile(archiveMetadataPath, archiveBytes, 0o644); err != nil {
		return fail(err)
	}

	now := clock().UTC()
	eventID := opts.EventID
	if eventID == nil {
		eventID = func(t time.Time) string { return "evt_" + t.Format("20060102T150405Z") }
	}
	if err := provenance.Append(path, provenance.Event{
		SchemaVersion: schemaVersion,
		ID:            eventID(now),
		Timestamp:     now.Format(time.RFC3339),
		Actor:         "rforge",
		Action:        "project.create",
		Target:        path,
		Inputs: map[string]any{
			"title":       title,
			"storageMode": "sqlite",
		},
		Outputs: map[string]any{
			"manifest":   "rforge.project.toml",
			"lockfile":   "rforge.lock.json",
			"provenance": "provenance/events.jsonl",
		},
		Warnings: []string{},
	}); err != nil {
		return fail(err)
	}

	return Project{
		Path:                path,
		Title:               title,
		StorageMode:         "sqlite",
		SchemaVersion:       schemaVersion,
		ManifestPath:        manifestPath,
		LockfilePath:        lockfilePath,
		ProvenancePath:      provenancePath,
		StoragePath:         storagePath,
		ArchiveMetadataPath: archiveMetadataPath,
	}, nil
}

// DiscoverAssets finds pre-existing local research assets and records discovery Provenance without importing them.
func DiscoverAssets(repoRoot, projectPath string) ([]Asset, error) {
	assets := []Asset{}
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, err
	}
	absProject, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}
	if err := filepath.WalkDir(absRepo, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == filepath.Join(absRepo, ".git") || path == absProject {
				return filepath.SkipDir
			}
			return nil
		}
		kind, ok := assetKind(path)
		if !ok {
			return nil
		}
		rel, err := filepath.Rel(absRepo, path)
		if err != nil {
			return err
		}
		assets = append(assets, Asset{Path: filepath.ToSlash(rel), Kind: kind, Imported: false})
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Path < assets[j].Path })
	unchanged, err := lastDiscoveryMatches(projectPath, assets)
	if err != nil {
		return nil, err
	}
	if unchanged {
		return assets, nil
	}
	now := time.Now().UTC()
	if err := provenance.Append(projectPath, provenance.Event{
		SchemaVersion: schemaVersion,
		ID:            "evt_" + now.Format("20060102T150405Z"),
		Timestamp:     now.Format(time.RFC3339),
		Actor:         "rforge",
		Action:        "project.assets.discover",
		Target:        repoRoot,
		Inputs: map[string]any{
			"repoRoot": repoRoot,
		},
		Outputs: map[string]any{
			"assetCount": len(assets),
			"assets":     assets,
		},
		Warnings: []string{},
	}); err != nil {
		return nil, err
	}
	return assets, nil
}

func lastDiscoveryMatches(projectPath string, assets []Asset) (bool, error) {
	return provenance.LastOutputEquals(projectPath, "project.assets.discover", "assets", assets)
}

func assetKind(path string) (string, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf":
		return "pdf", true
	case ".bib":
		return "bibliography", true
	case ".ris":
		return "bibliography", true
	case ".md":
		return "note", true
	default:
		return "", false
	}
}

// ValidatePath rejects empty paths and parent-directory traversal segments.
func ValidatePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("project path is required")
	}
	for _, part := range strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' }) {
		if part == ".." {
			return fmt.Errorf("project path must not contain parent traversal")
		}
	}
	return nil
}

// ReadEvents reads provenance events from a ResearchForge project.
func ReadEvents(path string) ([]Event, error) {
	return provenance.Read(path)
}

// Inspect reads a ResearchForge project workspace.
func Inspect(path string) (Project, error) {
	manifestBytes, err := os.ReadFile(filepath.Join(path, "rforge.project.toml"))
	if err != nil {
		return Project{}, err
	}
	values := parseSimpleTOML(string(manifestBytes))
	return Project{
		Path:                path,
		Title:               values["title"],
		StorageMode:         values["storage_mode"],
		SchemaVersion:       values["schema_version"],
		ManifestPath:        filepath.Join(path, "rforge.project.toml"),
		LockfilePath:        filepath.Join(path, "rforge.lock.json"),
		ProvenancePath:      filepath.Join(path, "provenance", "events.jsonl"),
		StoragePath:         filepath.Join(path, "data", "rforge.sqlite"),
		ArchiveMetadataPath: filepath.Join(path, "rforge.archive.json"),
	}, nil
}

// List reads ResearchForge projects directly under root.
func List(root string) ([]Project, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	projects := []Project{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		proj, err := Inspect(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		projects = append(projects, proj)
	}
	sort.Slice(projects, func(i, j int) bool { return projects[i].Path < projects[j].Path })
	return projects, nil
}

func parseSimpleTOML(content string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, "\"")
		values[key] = value
	}
	return values
}
