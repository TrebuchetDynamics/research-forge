package documents

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

// DocumentAssetInput is caller-provided metadata for a local document asset.
type DocumentAssetInput struct {
	PaperID           string
	AcquisitionSource string
	License           string
	OAStatus          string
	LocalPath         string
	MIMEType          string
	LocalOnly         bool
}

// DocumentAsset records provenance and policy metadata for a document file.
type DocumentAsset struct {
	SchemaVersion     string
	PaperID           string
	AcquisitionSource string
	License           string
	OAStatus          string
	ChecksumSHA256    string
	LocalPath         string
	MIMEType          string
	LocalOnly         bool
}

// NewDocumentAsset validates metadata and computes the local file checksum.
func NewDocumentAsset(input DocumentAssetInput) (DocumentAsset, error) {
	if strings.TrimSpace(input.PaperID) == "" {
		return DocumentAsset{}, fmt.Errorf("paper id is required")
	}
	if strings.TrimSpace(input.LocalPath) == "" {
		return DocumentAsset{}, fmt.Errorf("local path is required")
	}
	checksum, err := checksumFile(input.LocalPath)
	if err != nil {
		return DocumentAsset{}, err
	}
	return DocumentAsset{
		SchemaVersion:     "1",
		PaperID:           strings.TrimSpace(input.PaperID),
		AcquisitionSource: strings.TrimSpace(input.AcquisitionSource),
		License:           strings.TrimSpace(input.License),
		OAStatus:          strings.TrimSpace(input.OAStatus),
		ChecksumSHA256:    checksum,
		LocalPath:         strings.TrimSpace(input.LocalPath),
		MIMEType:          strings.TrimSpace(input.MIMEType),
		LocalOnly:         input.LocalOnly,
	}, nil
}

// GuardExport prevents accidental export of restricted or local-only assets.
func GuardExport(asset DocumentAsset) error {
	if asset.LocalOnly {
		return fmt.Errorf("local-only document assets cannot be exported")
	}
	if strings.TrimSpace(asset.License) == "" || strings.EqualFold(asset.OAStatus, "closed") {
		return fmt.Errorf("document asset is not legal open access for export")
	}
	return nil
}

// OpenAccessMetadata is the subset of OA lookup metadata needed for legal PDF selection.
type OpenAccessMetadata struct {
	OpenAccess bool
	OAStatus   string
	License    string
	PDFURL     string
}

// SelectLegalPDFURL returns a PDF URL only when metadata indicates legal open access.
func SelectLegalPDFURL(metadata OpenAccessMetadata) (string, error) {
	if !metadata.OpenAccess || strings.TrimSpace(metadata.License) == "" || strings.TrimSpace(metadata.PDFURL) == "" {
		return "", fmt.Errorf("no legal open-access PDF URL available")
	}
	return strings.TrimSpace(metadata.PDFURL), nil
}

func checksumFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
