// Package filetxn provides staged replacement for regular files.
package filetxn

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Output describes one file in an all-or-rollback replacement.
type Output struct {
	Path string
	Data []byte
	Mode os.FileMode
}

type snapshot struct {
	path    string
	data    []byte
	mode    os.FileMode
	existed bool
}

// ReplaceAll validates every target before changing any of them, then replaces
// each file. If a later replacement fails, earlier files are restored.
func ReplaceAll(outputs []Output) error {
	return replaceAll(outputs, nil)
}

// ReplaceAllThen replaces every output, runs commit, and restores every output
// to its prior state if commit fails.
func ReplaceAllThen(outputs []Output, commit func() error) error {
	if commit == nil {
		return fmt.Errorf("file transaction commit callback is required")
	}
	return replaceAll(outputs, commit)
}

func replaceAll(outputs []Output, commit func() error) error {
	prepared := append([]Output(nil), outputs...)
	snapshots := make([]snapshot, 0, len(prepared))
	seen := make(map[string]struct{}, len(prepared))
	for index := range prepared {
		output := &prepared[index]
		if output.Path == "" {
			return fmt.Errorf("file transaction target path is required")
		}
		cleanPath := filepath.Clean(output.Path)
		if _, exists := seen[cleanPath]; exists {
			return fmt.Errorf("duplicate file transaction target: %s", output.Path)
		}
		seen[cleanPath] = struct{}{}
		info, err := os.Lstat(output.Path)
		if err != nil {
			if os.IsNotExist(err) {
				snapshots = append(snapshots, snapshot{path: output.Path})
				continue
			}
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("file transaction target is not a regular file: %s", output.Path)
		}
		data, err := os.ReadFile(output.Path)
		if err != nil {
			return err
		}
		output.Mode = info.Mode().Perm()
		snapshots = append(snapshots, snapshot{path: output.Path, data: data, mode: info.Mode().Perm(), existed: true})
	}

	for index, output := range prepared {
		if err := Replace(output.Path, output.Data, output.Mode); err != nil {
			if rollbackErr := restore(snapshots[:index]); rollbackErr != nil {
				return fmt.Errorf("replace files: %w; rollback: %v", err, rollbackErr)
			}
			return err
		}
	}
	if commit != nil {
		if err := commit(); err != nil {
			if rollbackErr := restore(snapshots); rollbackErr != nil {
				return fmt.Errorf("commit file transaction: %w; rollback: %v", err, rollbackErr)
			}
			return err
		}
	}
	return nil
}

func restore(snapshots []snapshot) error {
	var failures []error
	for index := len(snapshots) - 1; index >= 0; index-- {
		snapshot := snapshots[index]
		var err error
		if snapshot.existed {
			err = Replace(snapshot.path, snapshot.data, snapshot.mode)
		} else if info, lstatErr := os.Lstat(snapshot.path); lstatErr == nil {
			if !info.Mode().IsRegular() {
				err = fmt.Errorf("rollback target is not a regular file: %s", snapshot.path)
			} else {
				err = os.Remove(snapshot.path)
			}
		} else if !os.IsNotExist(lstatErr) {
			err = lstatErr
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", snapshot.path, err))
		}
	}
	if len(failures) != 0 {
		return fmt.Errorf("restore file transaction: %v", failures)
	}
	return nil
}

// Replace writes data to a sibling staging file and installs it at path.
// Existing regular files are restored if installing the staged file fails.
func Replace(path string, data []byte, mode os.FileMode) error {
	return ReplaceFromReader(path, bytes.NewReader(data), mode)
}

// ReplaceFromReader streams source to a sibling staging file and installs it
// at path only after the complete stream has been written and closed.
func ReplaceFromReader(path string, source io.Reader, mode os.FileMode) error {
	if source == nil {
		return fmt.Errorf("file transaction source is required")
	}
	if err := validateParent(path); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("file transaction target is not a regular file: %s", path)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	staging, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".rforge-stage-*")
	if err != nil {
		return err
	}
	stagingPath := staging.Name()
	defer os.Remove(stagingPath)
	if err := staging.Chmod(mode); err != nil {
		_ = staging.Close()
		return err
	}
	if _, err := io.Copy(staging, source); err != nil {
		_ = staging.Close()
		return err
	}
	if err := staging.Close(); err != nil {
		return err
	}
	return install(stagingPath, path)
}

func validateParent(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("file transaction parent is not a directory: %s", dir)
	}
	return nil
}

func install(stagingPath, path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.Rename(stagingPath, path)
		}
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("file transaction target is not a regular file: %s", path)
	}
	backup, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".rforge-backup-*")
	if err != nil {
		return err
	}
	backupPath := backup.Name()
	if err := backup.Close(); err != nil {
		_ = os.Remove(backupPath)
		return err
	}
	if err := os.Remove(backupPath); err != nil {
		_ = os.RemoveAll(backupPath)
		return err
	}
	if err := os.Rename(path, backupPath); err != nil {
		return err
	}
	if err := os.Rename(stagingPath, path); err != nil {
		if restoreErr := os.Rename(backupPath, path); restoreErr != nil {
			return fmt.Errorf("replace file: %w; restore prior file from %s: %v", err, backupPath, restoreErr)
		}
		return err
	}
	_ = os.Remove(backupPath)
	return nil
}
