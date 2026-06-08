package project

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/storage"
)

const schemaVersion = "1"

// Project is a local ResearchForge workspace.
type Project struct {
	Path           string
	Title          string
	StorageMode    string
	SchemaVersion  string
	ManifestPath   string
	LockfilePath   string
	ProvenancePath string
	StoragePath    string
}

// Asset is a pre-existing local research asset discovered before import.
type Asset struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Imported bool   `json:"imported"`
}

// Event is a provenance event recorded in a ResearchForge project.
type Event struct {
	SchemaVersion string         `json:"schemaVersion"`
	ID            string         `json:"id"`
	Timestamp     string         `json:"timestamp"`
	Actor         string         `json:"actor"`
	Action        string         `json:"action"`
	Target        string         `json:"target"`
	Inputs        map[string]any `json:"inputs"`
	Outputs       map[string]any `json:"outputs"`
	Warnings      []string       `json:"warnings"`
}

// CreateOptions configures project creation.
type CreateOptions struct {
	Title   string
	Clock   func() time.Time
	EventID func(time.Time) string
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

	if err := os.MkdirAll(filepath.Join(path, "provenance"), 0o755); err != nil {
		return Project{}, err
	}
	if err := os.MkdirAll(filepath.Join(path, "data"), 0o755); err != nil {
		return Project{}, err
	}

	store, err := storage.Initialize(storagePath)
	if err != nil {
		return Project{}, err
	}
	if err := store.Close(); err != nil {
		return Project{}, err
	}

	manifest := fmt.Sprintf("schema_version = %q\ntitle = %q\nstorage_mode = %q\n", schemaVersion, title, "sqlite")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		return Project{}, err
	}

	lock := map[string]any{
		"schemaVersion": schemaVersion,
		"createdAt":     clock().UTC().Format(time.RFC3339),
		"tools":         map[string]any{},
	}
	lockBytes, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return Project{}, err
	}
	lockBytes = append(lockBytes, '\n')
	if err := os.WriteFile(lockfilePath, lockBytes, 0o644); err != nil {
		return Project{}, err
	}

	now := clock().UTC()
	eventID := opts.EventID
	if eventID == nil {
		eventID = func(t time.Time) string { return "evt_" + t.Format("20060102T150405Z") }
	}
	event := map[string]any{
		"schemaVersion": schemaVersion,
		"id":            eventID(now),
		"timestamp":     now.Format(time.RFC3339),
		"actor":         "rforge",
		"action":        "project.create",
		"target":        path,
		"inputs": map[string]any{
			"title":       title,
			"storageMode": "sqlite",
		},
		"outputs": map[string]any{
			"manifest":   "rforge.project.toml",
			"lockfile":   "rforge.lock.json",
			"provenance": "provenance/events.jsonl",
		},
		"warnings": []string{},
	}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return Project{}, err
	}
	eventBytes = append(eventBytes, '\n')
	if err := os.WriteFile(provenancePath, eventBytes, 0o644); err != nil {
		return Project{}, err
	}

	return Project{
		Path:           path,
		Title:          title,
		StorageMode:    "sqlite",
		SchemaVersion:  schemaVersion,
		ManifestPath:   manifestPath,
		LockfilePath:   lockfilePath,
		ProvenancePath: provenancePath,
		StoragePath:    storagePath,
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
	if err := appendEvent(projectPath, map[string]any{
		"schemaVersion": schemaVersion,
		"id":            "evt_" + time.Now().UTC().Format("20060102T150405Z"),
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"actor":         "rforge",
		"action":        "project.assets.discover",
		"target":        repoRoot,
		"inputs": map[string]any{
			"repoRoot": repoRoot,
		},
		"outputs": map[string]any{
			"assetCount": len(assets),
			"assets":     assets,
		},
		"warnings": []string{},
	}); err != nil {
		return nil, err
	}
	return assets, nil
}

func lastDiscoveryMatches(projectPath string, assets []Asset) (bool, error) {
	events, err := ReadEvents(projectPath)
	if err != nil {
		return false, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.Action != "project.assets.discover" {
			continue
		}
		previous, ok := event.Outputs["assets"]
		if !ok {
			return false, nil
		}
		return sameAssets(assetsFromEvent(previous), assets), nil
	}
	return false, nil
}

func assetsFromEvent(raw any) []Asset {
	rawAssets, ok := raw.([]any)
	if !ok {
		return nil
	}
	assets := make([]Asset, 0, len(rawAssets))
	for _, rawAsset := range rawAssets {
		values, ok := rawAsset.(map[string]any)
		if !ok {
			continue
		}
		asset := Asset{}
		if value, ok := values["path"].(string); ok {
			asset.Path = value
		}
		if value, ok := values["kind"].(string); ok {
			asset.Kind = value
		}
		if value, ok := values["imported"].(bool); ok {
			asset.Imported = value
		}
		assets = append(assets, asset)
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Path < assets[j].Path })
	return assets
}

func sameAssets(a, b []Asset) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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

func appendEvent(projectPath string, event map[string]any) error {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	eventBytes = append(eventBytes, '\n')
	file, err := os.OpenFile(filepath.Join(projectPath, "provenance", "events.jsonl"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(eventBytes)
	return err
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
	file, err := os.Open(filepath.Join(path, "provenance", "events.jsonl"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	events := []Event{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

// Inspect reads a ResearchForge project workspace.
func Inspect(path string) (Project, error) {
	manifestBytes, err := os.ReadFile(filepath.Join(path, "rforge.project.toml"))
	if err != nil {
		return Project{}, err
	}
	values := parseSimpleTOML(string(manifestBytes))
	return Project{
		Path:           path,
		Title:          values["title"],
		StorageMode:    values["storage_mode"],
		SchemaVersion:  values["schema_version"],
		ManifestPath:   filepath.Join(path, "rforge.project.toml"),
		LockfilePath:   filepath.Join(path, "rforge.lock.json"),
		ProvenancePath: filepath.Join(path, "provenance", "events.jsonl"),
		StoragePath:    filepath.Join(path, "data", "rforge.sqlite"),
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
