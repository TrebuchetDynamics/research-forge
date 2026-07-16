package filetxn

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// OpenRegular opens path for reading only when both the preflight target and
// opened file are the same regular file. The caller owns the returned file.
func OpenRegular(path string) (*os.File, error) {
	if err := validateParent(path); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("read target is not a regular file: %s", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	openedInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		_ = file.Close()
		return nil, fmt.Errorf("read target changed while opening: %s", path)
	}
	return file, nil
}

// OpenRegularInRoot opens name only when it remains beneath root and the
// opened file is the same regular file visible at that project-relative path.
// The caller owns the returned file.
func OpenRegularInRoot(root, name string) (*os.File, error) {
	file, err := os.OpenInRoot(root, name)
	if err != nil {
		return nil, err
	}
	openedInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	path := filepath.Join(root, name)
	info, err := os.Lstat(path)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() || !openedInfo.Mode().IsRegular() || !os.SameFile(info, openedInfo) {
		_ = file.Close()
		return nil, fmt.Errorf("read target is not the same regular file beneath root: %s", path)
	}
	return file, nil
}

// ReadRegular reads path only after OpenRegular validates its identity.
func ReadRegular(path string) ([]byte, error) {
	file, err := OpenRegular(path)
	if err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		return nil, readErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return data, nil
}
