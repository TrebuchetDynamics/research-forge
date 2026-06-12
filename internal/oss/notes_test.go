package oss

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteStudyNoteTemplate(t *testing.T) {
	projectPath := t.TempDir()
	path, err := WriteStudyNote(projectPath, "owner/repo", "architecture")
	if err != nil {
		t.Fatalf("WriteStudyNote returned error: %v", err)
	}
	if path != filepath.Join(projectPath, "opensource", "notes", "owner", "repo", "architecture.md") {
		t.Fatalf("path = %q", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	text := string(data)
	for _, want := range []string{"# owner/repo — architecture", "## Summary", "## License and provenance", "Do not copy external source code"} {
		if !strings.Contains(text, want) {
			t.Fatalf("note missing %q:\n%s", want, text)
		}
	}
}
