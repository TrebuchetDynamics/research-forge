package oss

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TopicScan records metadata for a topic-oriented scan of an OSS repository.
type TopicScan struct {
	SchemaVersion string
	Repository    string
	Topic         string
	Path          string
}

// WriteTopicScan writes deterministic scan metadata without copying external source code.
func WriteTopicScan(projectPath, name, topic string) (TopicScan, error) {
	study, err := NewRepositoryStudy(RepositoryStudyInput{Name: name})
	if err != nil {
		return TopicScan{}, err
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return TopicScan{}, fmt.Errorf("scan topic is required")
	}
	dir := filepath.Join(projectPath, "opensource", "scans", study.Owner, study.Repo)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return TopicScan{}, err
	}
	scan := TopicScan{SchemaVersion: "1", Repository: study.Name, Topic: topic, Path: filepath.Join(dir, safeNoteName(topic)+".json")}
	data, err := json.MarshalIndent(scan, "", "  ")
	if err != nil {
		return TopicScan{}, err
	}
	data = append(data, '\n')
	return scan, os.WriteFile(scan.Path, data, 0o644)
}

// AreaReport is a markdown OSS study report for one research area.
type AreaReport struct {
	Area     string
	Markdown string
}

// BuildAreaReport summarizes registered repositories for a research area.
func BuildAreaReport(projectPath string, registry Registry, area string) (AreaReport, error) {
	items, err := registry.List()
	if err != nil {
		return AreaReport{}, err
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "# OSS report: %s\n\n", area)
	for _, item := range items {
		if item.Area == area {
			fmt.Fprintf(&builder, "- %s (%s)\n", item.Name, item.ClonePath)
		}
	}
	return AreaReport{Area: area, Markdown: builder.String()}, nil
}
