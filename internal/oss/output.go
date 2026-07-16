package oss

import (
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

func writeOutput(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return filetxn.Replace(path, data, 0o644)
}
