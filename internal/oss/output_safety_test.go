package oss

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestOSSWritersDoNotWriteThroughSymlinkedDestinations(t *testing.T) {
	tests := []struct {
		name       string
		targetPath func(string) string
		write      func(string, string) error
	}{
		{
			name: "study note",
			targetPath: func(root string) string {
				return filepath.Join(root, "opensource", "notes", "owner", "repo", "safety.md")
			},
			write: func(root, _ string) error {
				_, err := WriteStudyNote(root, "owner/repo", "safety")
				return err
			},
		},
		{
			name: "topic scan",
			targetPath: func(root string) string {
				return filepath.Join(root, "opensource", "scans", "owner", "repo", "safety.json")
			},
			write: func(root, _ string) error {
				_, err := WriteTopicScan(root, "owner/repo", "safety")
				return err
			},
		},
		{
			name:       "inventory manifest",
			targetPath: func(root string) string { return filepath.Join(root, "inventory", "manifest.json") },
			write: func(_ string, target string) error {
				return SaveInventoryManifest(target, InventoryManifest{SchemaVersion: "1"})
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			targetPath := tc.targetPath(root)
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				t.Fatalf("create output directory: %v", err)
			}
			outsidePath := filepath.Join(t.TempDir(), "outside-output")
			outsideBefore := []byte("outside OSS output\n")
			if err := os.WriteFile(outsidePath, outsideBefore, 0o600); err != nil {
				t.Fatalf("write outside output: %v", err)
			}
			if err := os.Symlink(outsidePath, targetPath); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}

			if err := tc.write(root, targetPath); err == nil {
				t.Errorf("writer succeeded through symlink")
			}
			outsideAfter, err := os.ReadFile(outsidePath)
			if err != nil {
				t.Fatalf("read outside output: %v", err)
			}
			if !bytes.Equal(outsideAfter, outsideBefore) {
				t.Errorf("writer changed outside output:\n got: %s\nwant: %s", outsideAfter, outsideBefore)
			}
			info, err := os.Stat(outsidePath)
			if err != nil {
				t.Fatalf("stat outside output: %v", err)
			}
			if got := info.Mode().Perm(); got != 0o600 {
				t.Errorf("outside output mode = %o, want 600", got)
			}
		})
	}
}
