package oss

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteTopicScanMetadataAndAreaReport(t *testing.T) {
	projectPath := t.TempDir()
	registry, err := OpenRegistry(filepath.Join(projectPath, "data", "oss.json"))
	if err != nil {
		t.Fatalf("OpenRegistry returned error: %v", err)
	}
	study, _ := NewRepositoryStudy(RepositoryStudyInput{Name: "owner/repo", Area: "literature tooling"})
	if err := registry.Add(study); err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	scan, err := WriteTopicScan(projectPath, "owner/repo", "deduplication")
	if err != nil {
		t.Fatalf("WriteTopicScan returned error: %v", err)
	}
	if scan.Topic != "deduplication" || scan.Repository != "owner/repo" {
		t.Fatalf("scan = %#v", scan)
	}
	if _, err := os.Stat(scan.Path); err != nil {
		t.Fatalf("missing scan file: %v", err)
	}
	report, err := BuildAreaReport(projectPath, registry, "literature tooling")
	if err != nil {
		t.Fatalf("BuildAreaReport returned error: %v", err)
	}
	if !strings.Contains(report.Markdown, "# OSS report: literature tooling") || !strings.Contains(report.Markdown, "owner/repo") {
		t.Fatalf("report = %s", report.Markdown)
	}
}
