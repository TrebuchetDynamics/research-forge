package project

import (
	"archive/tar"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestArchiveSourceValidation(t *testing.T) {
	for _, sourceKind := range []string{"symlink", "symlink-trailing", "symlink-dot", "file", "missing", "empty"} {
		for _, destination := range []string{"absent", "existing"} {
			t.Run(sourceKind+"/"+destination, func(t *testing.T) {
				root := t.TempDir()
				target := filepath.Join(root, "target")
				// The usable output directory is inside the link target, not the lexical link path.
				outputDir := filepath.Join(target, "backups")
				if err := os.MkdirAll(outputDir, 0o755); err != nil {
					t.Fatal(err)
				}
				payload := filepath.Join(target, "payload.txt")
				file := filepath.Join(root, "file.txt")
				const contents = "source contents must not appear in errors"
				for _, path := range []string{payload, file} {
					if err := os.WriteFile(path, []byte(contents), 0o640); err != nil {
						t.Fatal(err)
					}
				}
				source := file
				link := filepath.Join(root, "link")
				switch sourceKind {
				case "symlink", "symlink-trailing", "symlink-dot":
					if err := os.Symlink(target, link); err != nil {
						t.Skipf("symlink unsupported: %v", err)
					}
					source = link
					if sourceKind != "symlink" {
						source += string(filepath.Separator)
					}
					if sourceKind == "symlink-dot" {
						source += "."
					}
				case "missing":
					source = filepath.Join(root, "missing")
				case "empty":
					source = ""
				}
				archivePath := filepath.Join(outputDir, "backup.tar")
				if destination == "existing" {
					if err := os.WriteFile(archivePath, []byte("prior archive"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				err := Archive(source, archivePath)
				if err == nil {
					t.Error("Archive accepted a non-directory source")
				} else {
					if !strings.Contains(err.Error(), "directory") && (source == "" || !strings.Contains(err.Error(), source)) {
						t.Errorf("error lacks source/directory context: %v", err)
					}
					if strings.Contains(err.Error(), contents) {
						t.Errorf("error contains source contents: %v", err)
					}
				}
				if sourceKind == "missing" && !errors.Is(err, os.ErrNotExist) {
					t.Errorf("missing source cause lost: %v", err)
				}
				wantEntries := 0
				if destination == "existing" {
					wantEntries = 1
					if data, err := os.ReadFile(archivePath); err != nil || string(data) != "prior archive" {
						t.Errorf("prior archive changed: %q, %v", data, err)
					}
					if info, err := os.Stat(archivePath); err != nil || info.Mode().Perm() != 0o600 {
						t.Errorf("prior archive permissions changed: %v, %v", info, err)
					}
				} else if _, err := os.Lstat(archivePath); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("absent archive created: %v", err)
				}
				if entries, err := os.ReadDir(outputDir); err != nil || len(entries) != wantEntries {
					t.Errorf("output inventory = %v, %v; want %d entries", entries, err, wantEntries)
				}
				for _, path := range []string{payload, file} {
					if data, err := os.ReadFile(path); err != nil || string(data) != contents {
						t.Errorf("source changed: %s = %q, %v", path, data, err)
					}
				}
				if strings.HasPrefix(sourceKind, "symlink") {
					if got, err := os.Readlink(link); err != nil || got != target {
						t.Errorf("source link changed: %q, %v", got, err)
					}
				}
			})
		}
	}

	for _, spelling := range []string{"directory", "trailing-separator", "empty-directory"} {
		t.Run(spelling, func(t *testing.T) {
			source := t.TempDir() // No project manifest is required.
			want := map[string]string{}
			if spelling != "empty-directory" {
				want = map[string]string{"nested/file.txt": "nested bytes", "file with spaces.txt": "spaced bytes"}
				for name, contents := range want {
					path := filepath.Join(source, filepath.FromSlash(name))
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte(contents), 0o640); err != nil {
						t.Fatal(err)
					}
				}
			}
			if spelling == "trailing-separator" {
				source += string(filepath.Separator)
			}
			archivePath := filepath.Join(t.TempDir(), "backup.tar")
			if err := Archive(source, archivePath); err != nil {
				t.Fatal(err)
			}
			archive, err := os.Open(archivePath)
			if err != nil {
				t.Fatal(err)
			}
			defer archive.Close()
			reader := tar.NewReader(archive)
			got := map[string]string{}
			for {
				header, err := reader.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(reader)
				if err != nil {
					t.Fatal(err)
				}
				got[header.Name] = string(data)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("archive members = %v, want %v", got, want)
			}
			restored := t.TempDir()
			if err := Restore(archivePath, restored); err != nil {
				t.Fatal(err)
			}
			for name, contents := range want {
				if data, err := os.ReadFile(filepath.Join(restored, filepath.FromSlash(name))); err != nil || string(data) != contents {
					t.Errorf("restored %s = %q, %v", name, data, err)
				}
			}
		})
	}
}
