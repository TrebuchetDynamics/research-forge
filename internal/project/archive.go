package project

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

func Archive(projectPath, archivePath string) error {
	if err := guardArchiveOutputPath(projectPath, archivePath); err != nil {
		return err
	}
	archiveAbs, err := filepath.Abs(archivePath)
	if err != nil {
		return err
	}
	out, err := os.CreateTemp(filepath.Dir(archiveAbs), "."+filepath.Base(archiveAbs)+".rforge-stage-*")
	if err != nil {
		return err
	}
	stagingPath := out.Name()
	defer os.Remove(stagingPath)
	if err := out.Chmod(0o644); err != nil {
		_ = out.Close()
		return err
	}
	archiveErr := archiveTo(out, projectPath)
	closeErr := out.Close()
	if archiveErr != nil {
		return archiveErr
	}
	if closeErr != nil {
		return closeErr
	}
	return replaceArchiveOutput(stagingPath, archiveAbs)
}

func replaceArchiveOutput(stagingPath, archivePath string) error {
	info, err := os.Lstat(archivePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return os.Rename(stagingPath, archivePath)
	}
	if !info.Mode().IsRegular() {
		return os.ErrPermission
	}
	backupPath, err := os.CreateTemp(filepath.Dir(archivePath), "."+filepath.Base(archivePath)+".rforge-backup-*")
	if err != nil {
		return err
	}
	if err := backupPath.Close(); err != nil {
		_ = os.Remove(backupPath.Name())
		return err
	}
	backupName := backupPath.Name()
	if err := os.Remove(backupName); err != nil {
		_ = os.RemoveAll(backupName)
		return err
	}
	if err := os.Rename(archivePath, backupName); err != nil {
		return err
	}
	if err := os.Rename(stagingPath, archivePath); err != nil {
		if restoreErr := os.Rename(backupName, archivePath); restoreErr != nil {
			return fmt.Errorf("replace archive: %w; restore prior archive from %s: %v", err, backupName, restoreErr)
		}
		return err
	}
	_ = os.Remove(backupName)
	return nil
}

// archiveTo writes a tar archive of projectPath to w, returning any write or
// close error. Close is where the tar trailer is flushed, so a write failure
// there must not be swallowed or the resulting archive is silently corrupt.
func archiveTo(w io.Writer, projectPath string) error {
	tw := tar.NewWriter(w)
	walkErr := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(projectPath, path)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, in)
		closeErr := in.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	closeErr := tw.Close()
	if walkErr != nil {
		return walkErr
	}
	return closeErr
}

func guardArchiveOutputPath(projectPath, archivePath string) error {
	projectAbs, err := filepath.Abs(projectPath)
	if err != nil {
		return err
	}
	archiveAbs, err := filepath.Abs(archivePath)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(archiveAbs); err == nil && !info.Mode().IsRegular() {
		return os.ErrPermission
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	rel, err := filepath.Rel(projectAbs, archiveAbs)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return os.ErrPermission
	}
	return nil
}

func Restore(archivePath, destination string) error {
	in, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer in.Close()
	stagingRoot, err := os.MkdirTemp("", "rforge-restore-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stagingRoot)
	if err := extractArchive(tar.NewReader(in), stagingRoot); err != nil {
		return err
	}
	return installRestore(stagingRoot, destination)
}

func extractArchive(tr *tar.Reader, stagingRoot string) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		clean := filepath.Clean(hdr.Name)
		if filepath.IsAbs(hdr.Name) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return os.ErrPermission
		}
		target := filepath.Join(stagingRoot, clean)
		if err := guardRestoreTarget(stagingRoot, target); err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, hdr.FileInfo().Mode()); err != nil {
				return err
			}
			continue
		case tar.TypeReg, tar.TypeRegA:
		default:
			return os.ErrPermission
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := filetxn.ReplaceFromReader(target, tr, hdr.FileInfo().Mode()); err != nil {
			return err
		}
	}
	return nil
}

func installRestore(stagingRoot, destination string) error {
	if err := preflightRestore(stagingRoot, destination); err != nil {
		return err
	}
	transaction, err := captureRestoreTransaction(stagingRoot, destination)
	if err != nil {
		return err
	}
	defer os.RemoveAll(transaction.rollbackRoot)
	installErr := filepath.WalkDir(stagingRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == stagingRoot {
			return nil
		}
		relativePath, err := filepath.Rel(stagingRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relativePath)
		if err := guardRestoreTarget(destination, target); err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if !info.Mode().IsRegular() {
			return os.ErrPermission
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		source, err := os.Open(path)
		if err != nil {
			return err
		}
		replaceErr := filetxn.ReplaceFromReader(target, source, info.Mode())
		closeErr := source.Close()
		if replaceErr != nil {
			return replaceErr
		}
		return closeErr
	})
	if installErr != nil {
		return transaction.rollback(installErr)
	}
	return nil
}

func preflightRestore(stagingRoot, destination string) error {
	return filepath.WalkDir(stagingRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == stagingRoot {
			return nil
		}
		relativePath, err := filepath.Rel(stagingRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relativePath)
		if err := guardRestoreTarget(destination, target); err != nil {
			return err
		}
		stagedInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if !entry.IsDir() && !stagedInfo.Mode().IsRegular() {
			return os.ErrPermission
		}
		targetInfo, err := os.Lstat(target)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if entry.IsDir() != targetInfo.IsDir() {
			return fmt.Errorf("restore target type conflicts with archive entry: %s", target)
		}
		if !entry.IsDir() && !targetInfo.Mode().IsRegular() {
			return fmt.Errorf("restore target is not a regular file: %s", target)
		}
		return nil
	})
}

type restoreFileSnapshot struct {
	target     string
	backupPath string
	mode       os.FileMode
	existed    bool
}

type restoreInstallTransaction struct {
	rollbackRoot string
	files        []restoreFileSnapshot
	createdDirs  []string
}

func captureRestoreTransaction(stagingRoot, destination string) (restoreInstallTransaction, error) {
	rollbackRoot, err := os.MkdirTemp("", "rforge-restore-rollback-*")
	if err != nil {
		return restoreInstallTransaction{}, err
	}
	transaction := restoreInstallTransaction{rollbackRoot: rollbackRoot}
	createdDirs := map[string]struct{}{}
	fail := func(err error) (restoreInstallTransaction, error) {
		_ = os.RemoveAll(rollbackRoot)
		return restoreInstallTransaction{}, err
	}
	err = filepath.WalkDir(stagingRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == stagingRoot {
			return nil
		}
		relativePath, err := filepath.Rel(stagingRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relativePath)
		dir := filepath.Dir(target)
		if entry.IsDir() {
			dir = target
		}
		if err := collectMissingRestoreDirs(dir, createdDirs); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		snapshot := restoreFileSnapshot{target: target}
		info, err := os.Lstat(target)
		if os.IsNotExist(err) {
			transaction.files = append(transaction.files, snapshot)
			return nil
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("restore target is not a regular file: %s", target)
		}
		source, err := os.Open(target)
		if err != nil {
			return err
		}
		snapshot.backupPath = filepath.Join(rollbackRoot, fmt.Sprintf("%d.bak", len(transaction.files)))
		snapshot.mode = info.Mode()
		snapshot.existed = true
		copyErr := filetxn.ReplaceFromReader(snapshot.backupPath, source, snapshot.mode)
		closeErr := source.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		transaction.files = append(transaction.files, snapshot)
		return nil
	})
	if err != nil {
		return fail(err)
	}
	for dir := range createdDirs {
		transaction.createdDirs = append(transaction.createdDirs, dir)
	}
	sort.Slice(transaction.createdDirs, func(i, j int) bool {
		return len(transaction.createdDirs[i]) > len(transaction.createdDirs[j])
	})
	return transaction, nil
}

func collectMissingRestoreDirs(dir string, createdDirs map[string]struct{}) error {
	for {
		info, err := os.Lstat(dir)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("restore parent is not a directory: %s", dir)
			}
			return nil
		}
		if !os.IsNotExist(err) {
			return err
		}
		createdDirs[dir] = struct{}{}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

func (transaction restoreInstallTransaction) rollback(cause error) error {
	var failures []error
	for index := len(transaction.files) - 1; index >= 0; index-- {
		snapshot := transaction.files[index]
		var err error
		if snapshot.existed {
			var source *os.File
			source, err = os.Open(snapshot.backupPath)
			if err == nil {
				err = filetxn.ReplaceFromReader(snapshot.target, source, snapshot.mode)
				if closeErr := source.Close(); err == nil {
					err = closeErr
				}
			}
		} else {
			info, lstatErr := os.Lstat(snapshot.target)
			if lstatErr == nil {
				if !info.Mode().IsRegular() {
					err = fmt.Errorf("restore rollback target is not a regular file: %s", snapshot.target)
				} else {
					err = os.Remove(snapshot.target)
				}
			} else if !os.IsNotExist(lstatErr) {
				err = lstatErr
			}
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", snapshot.target, err))
		}
	}
	for _, dir := range transaction.createdDirs {
		if err := os.Remove(dir); err != nil && !os.IsNotExist(err) {
			failures = append(failures, fmt.Errorf("%s: %w", dir, err))
		}
	}
	if len(failures) != 0 {
		return fmt.Errorf("%w; roll back restore: %v", cause, errors.Join(failures...))
	}
	return cause
}

func guardRestoreTarget(destination, target string) error {
	destinationAbs, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	if info, err := os.Lstat(destinationAbs); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return os.ErrPermission
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(destinationAbs, targetAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return os.ErrPermission
	}
	current := destinationAbs
	for _, part := range strings.Split(filepath.Clean(rel), string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return os.ErrPermission
		}
	}
	return nil
}
