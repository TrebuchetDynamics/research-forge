package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const schemaVersion = "1"

// Project is a local ResearchForge workspace.
type Project struct {
	Path        string
	Title       string
	StorageMode string
}

// CreateOptions configures project creation.
type CreateOptions struct {
	Title string
	Clock func() time.Time
}

// Create initializes a ResearchForge project workspace.
func Create(path string, opts CreateOptions) (Project, error) {
	if strings.TrimSpace(path) == "" {
		return Project{}, fmt.Errorf("project path is required")
	}
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		return Project{}, fmt.Errorf("project title is required")
	}
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}

	if err := os.MkdirAll(filepath.Join(path, "provenance"), 0o755); err != nil {
		return Project{}, err
	}
	if err := os.MkdirAll(filepath.Join(path, "data"), 0o755); err != nil {
		return Project{}, err
	}

	manifest := fmt.Sprintf("schema_version = %q\ntitle = %q\nstorage_mode = %q\n", schemaVersion, title, "sqlite")
	if err := os.WriteFile(filepath.Join(path, "rforge.project.toml"), []byte(manifest), 0o644); err != nil {
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
	if err := os.WriteFile(filepath.Join(path, "rforge.lock.json"), lockBytes, 0o644); err != nil {
		return Project{}, err
	}

	now := clock().UTC()
	event := map[string]any{
		"schemaVersion": schemaVersion,
		"id":            "evt_" + now.Format("20060102T150405Z"),
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
	if err := os.WriteFile(filepath.Join(path, "provenance", "events.jsonl"), eventBytes, 0o644); err != nil {
		return Project{}, err
	}

	return Project{Path: path, Title: title, StorageMode: "sqlite"}, nil
}

// Inspect reads a ResearchForge project workspace.
func Inspect(path string) (Project, error) {
	manifestBytes, err := os.ReadFile(filepath.Join(path, "rforge.project.toml"))
	if err != nil {
		return Project{}, err
	}
	values := parseSimpleTOML(string(manifestBytes))
	return Project{
		Path:        path,
		Title:       values["title"],
		StorageMode: values["storage_mode"],
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
