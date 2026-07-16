package documents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImportLocalFile copies a manually supplied document into project-local storage as local-only.
func ImportLocalFile(projectPath, sourcePath, paperID string) (DocumentAsset, error) {
	if strings.TrimSpace(projectPath) == "" {
		return DocumentAsset{}, fmt.Errorf("project path is required")
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return DocumentAsset{}, err
	}
	destination := filepath.Join(projectPath, "documents", "local", filepath.Base(sourcePath))
	if err := writeDocumentFile(projectPath, destination, data, 0o644); err != nil {
		return DocumentAsset{}, err
	}
	return NewDocumentAsset(DocumentAssetInput{
		PaperID:           paperID,
		AcquisitionSource: "manual-local",
		OAStatus:          "local-only",
		LocalPath:         destination,
		MIMEType:          detectMIMEFromName(destination),
		LocalOnly:         true,
	})
}

func detectMIMEFromName(path string) string {
	if strings.EqualFold(filepath.Ext(path), ".pdf") {
		return "application/pdf"
	}
	return "application/octet-stream"
}
