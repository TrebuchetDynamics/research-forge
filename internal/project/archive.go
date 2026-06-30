package project

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func Archive(projectPath, archivePath string) error {
	if err := guardArchiveOutputPath(projectPath, archivePath); err != nil {
		return err
	}
	out, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	archiveErr := archiveTo(out, projectPath)
	closeErr := out.Close()
	if archiveErr != nil {
		return archiveErr
	}
	return closeErr
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
	if info, err := os.Lstat(archiveAbs); err == nil && info.Mode()&os.ModeSymlink != 0 {
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
	tr := tar.NewReader(in)
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
		target := filepath.Join(destination, clean)
		if err := guardRestoreTarget(destination, target); err != nil {
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
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
	return nil
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
