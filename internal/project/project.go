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
	Path        string
	Title       string
	StorageMode string
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

	if err := os.MkdirAll(filepath.Join(path, "provenance"), 0o755); err != nil {
		return Project{}, err
	}
	if err := os.MkdirAll(filepath.Join(path, "data"), 0o755); err != nil {
		return Project{}, err
	}

	store, err := storage.Initialize(filepath.Join(path, "data", "rforge.sqlite"))
	if err != nil {
		return Project{}, err
	}
	if err := store.Close(); err != nil {
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
	if err := os.WriteFile(filepath.Join(path, "provenance", "events.jsonl"), eventBytes, 0o644); err != nil {
		return Project{}, err
	}

	return Project{Path: path, Title: title, StorageMode: "sqlite"}, nil
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
