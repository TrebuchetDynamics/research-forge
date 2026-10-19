package research

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

// SearchConnector is the source operation needed by ranking. Adapters supply
// configured connectors, keeping credentials and transport policy out of research.
type SearchConnector interface {
	Search(context.Context, sources.SourceQuery) (sources.SourceResponse, error)
}

// BuildInfo identifies the engine that produced an artifact.
type BuildInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// Service runs research operations independently of any command or transport.
// Connector is required for Rank; Now and Build may be omitted for local use.
type Service struct {
	Connector func(string) (SearchConnector, bool)
	Now       func() time.Time
	Build     BuildInfo
}

func (s Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Error preserves the operation's machine-readable failure code for adapters.
type Error struct {
	Code     string
	ExitCode int
	Err      error
}

func (e *Error) Error() string             { return e.Err.Error() }
func (e *Error) Unwrap() error             { return e.Err }
func failure(code string, err error) error { return &Error{Code: code, ExitCode: 1, Err: err} }
func invalid(code, message string) error {
	return &Error{Code: code, ExitCode: 2, Err: fmt.Errorf("%s", message)}
}

func writeJSONArtifact(path string, value any, commit func() error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	outputs := []filetxn.Output{{Path: path, Data: append(data, '\n'), Mode: 0644}}
	if commit != nil {
		return filetxn.ReplaceAllThen(outputs, commit)
	}
	return filetxn.ReplaceAll(outputs)
}

func (s Service) recordRun(root, action string, metadata map[string]any, outputs []string, sidecar string) (provenance.Validation, error) {
	if _, ok := metadata["question"]; !ok {
		metadata["question"] = action
	}
	if _, ok := metadata["depth"]; !ok {
		metadata["depth"] = "standard"
	}
	if _, ok := metadata["sources"]; !ok {
		metadata["sources"] = []string{}
	}
	if _, ok := metadata["errors"]; !ok {
		metadata["errors"] = []string{}
	}
	metadata["rforge_version"] = s.buildInfo()
	return provenance.WriteRun(root, action, metadata, outputs, sidecar)
}

func (s Service) buildInfo() BuildInfo {
	build := s.Build
	if build.Version == "" {
		build.Version = "dev"
	}
	if build.Commit == "" {
		build.Commit = "unknown"
	}
	if build.Date == "" {
		build.Date = "unknown"
	}
	return build
}
