package oss

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var repoNamePattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// RepositoryStudyInput is caller-provided OSS repository study metadata.
type RepositoryStudyInput struct {
	Name            string
	Area            string
	RefreshInterval string
	Archived        bool
	Stale           bool
}

// OSSRepositoryStudy tracks one external open-source repository under study.
type OSSRepositoryStudy struct {
	SchemaVersion   string
	Name            string
	Owner           string
	Repo            string
	ClonePath       string
	Area            string
	RefreshInterval string
	Archived        bool
	Stale           bool
}

// NewRepositoryStudy validates owner/repo names and creates study metadata.
func NewRepositoryStudy(input RepositoryStudyInput) (OSSRepositoryStudy, error) {
	name := strings.TrimSpace(input.Name)
	if !repoNamePattern.MatchString(name) || strings.Contains(name, "..") || strings.HasSuffix(name, ".git") {
		return OSSRepositoryStudy{}, fmt.Errorf("repository name must be owner/repo")
	}
	parts := strings.Split(name, "/")
	return OSSRepositoryStudy{
		SchemaVersion:   "1",
		Name:            name,
		Owner:           parts[0],
		Repo:            parts[1],
		ClonePath:       filepath.Join("opensource", "clones", parts[0], parts[1]),
		Area:            strings.TrimSpace(input.Area),
		RefreshInterval: strings.TrimSpace(input.RefreshInterval),
		Archived:        input.Archived,
		Stale:           input.Stale,
	}, nil
}

// Registry stores OSSRepositoryStudy records in a local JSON file.
type Registry struct {
	path string
}

// OpenRegistry opens a JSON-backed OSS registry.
func OpenRegistry(path string) (Registry, error) {
	if strings.TrimSpace(path) == "" {
		return Registry{}, fmt.Errorf("oss registry path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Registry{}, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte("[]\n"), 0o644); err != nil {
			return Registry{}, err
		}
	} else if err != nil {
		return Registry{}, err
	}
	return Registry{path: path}, nil
}

// Add inserts a repository study if not already present.
func (r Registry) Add(study OSSRepositoryStudy) error {
	items, err := r.List()
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.Name == study.Name {
			return fmt.Errorf("oss repository already exists")
		}
	}
	items = append(items, study)
	return r.write(items)
}

// RefreshMetadata stores scheduled refresh, stale, and archived metadata for a repository.
type RefreshMetadata struct {
	RefreshInterval string
	Stale           bool
	Archived        bool
}

// RefreshMetadata updates registry metadata for one repository.
func (r Registry) RefreshMetadata(name string, metadata RefreshMetadata) error {
	items, err := r.List()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].Name == name {
			items[i].RefreshInterval = strings.TrimSpace(metadata.RefreshInterval)
			items[i].Stale = metadata.Stale
			items[i].Archived = metadata.Archived
			return r.write(items)
		}
	}
	return fmt.Errorf("oss repository not found")
}

// List returns repository studies sorted by name.
func (r Registry) List() ([]OSSRepositoryStudy, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil, err
	}
	var items []OSSRepositoryStudy
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (r Registry) write(items []OSSRepositoryStudy) error {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(r.path, data, 0o644)
}

// ResolveClonePath returns the safe local clone location for owner/repo.
func ResolveClonePath(projectPath, name string) (string, error) {
	study, err := NewRepositoryStudy(RepositoryStudyInput{Name: name})
	if err != nil {
		return "", err
	}
	root := filepath.Clean(filepath.Join(projectPath, "opensource", "clones"))
	resolved := filepath.Clean(filepath.Join(projectPath, study.ClonePath))
	if resolved != root && !strings.HasPrefix(resolved, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("clone path escapes opensource/clones")
	}
	return resolved, nil
}

// LicenseDetection describes a detected repository license file.
type LicenseDetection struct {
	Found bool
	Path  string
	Kind  string
}

// DetectLicenseFile detects common license files and a small set of license kinds.
func DetectLicenseFile(repoPath string) (LicenseDetection, error) {
	for _, name := range []string{"LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING"} {
		path := filepath.Join(repoPath, name)
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return LicenseDetection{}, err
		}
		return LicenseDetection{Found: true, Path: path, Kind: detectLicenseKind(string(data))}, nil
	}
	return LicenseDetection{Found: false}, nil
}

func detectLicenseKind(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "mit license"):
		return "MIT"
	case strings.Contains(lower, "apache license"):
		return "Apache"
	case strings.Contains(lower, "gnu general public license"):
		return "GPL"
	default:
		return "unknown"
	}
}
