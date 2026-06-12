package documents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDocumentAssetRecordsAcquisitionLicenseChecksumPathAndMIME(t *testing.T) {
	file := filepath.Join(t.TempDir(), "paper.pdf")
	if err := os.WriteFile(file, []byte("%PDF-1.4 artificial photosynthesis fixture"), 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}
	asset, err := NewDocumentAsset(DocumentAssetInput{
		PaperID:           "10.1000/example",
		AcquisitionSource: "unpaywall",
		License:           "cc-by",
		OAStatus:          "gold",
		LocalPath:         file,
		MIMEType:          "application/pdf",
	})
	if err != nil {
		t.Fatalf("NewDocumentAsset returned error: %v", err)
	}
	if asset.SchemaVersion != "1" || asset.PaperID != "10.1000/example" || asset.AcquisitionSource != "unpaywall" || asset.License != "cc-by" || asset.OAStatus != "gold" || asset.MIMEType != "application/pdf" {
		t.Fatalf("asset = %#v", asset)
	}
	if asset.ChecksumSHA256 == "" || asset.LocalPath != file {
		t.Fatalf("checksum/path = %#v", asset)
	}
}

func TestCopyrightGuardAllowsOnlyOpenAccessOrLocalOnly(t *testing.T) {
	openAsset := DocumentAsset{OAStatus: "gold", License: "cc-by", LocalPath: "paper.pdf"}
	if err := GuardExport(openAsset); err != nil {
		t.Fatalf("GuardExport(openAsset) returned error: %v", err)
	}
	restricted := DocumentAsset{OAStatus: "closed", License: "", LocalPath: "restricted.pdf"}
	if err := GuardExport(restricted); err == nil {
		t.Fatalf("GuardExport(restricted) returned nil error")
	}
	localOnly := DocumentAsset{OAStatus: "local-only", LocalOnly: true, LocalPath: "private.pdf"}
	if err := GuardExport(localOnly); err == nil {
		t.Fatalf("GuardExport(localOnly) returned nil error")
	}
}

func TestSelectLegalPDFURLFromUnpaywallMetadata(t *testing.T) {
	url, err := SelectLegalPDFURL(OpenAccessMetadata{OpenAccess: true, OAStatus: "green", License: "cc-by", PDFURL: "https://example.org/paper.pdf"})
	if err != nil {
		t.Fatalf("SelectLegalPDFURL returned error: %v", err)
	}
	if url != "https://example.org/paper.pdf" {
		t.Fatalf("url = %q", url)
	}
	if _, err := SelectLegalPDFURL(OpenAccessMetadata{OpenAccess: false, PDFURL: "https://example.org/paper.pdf"}); err == nil {
		t.Fatalf("SelectLegalPDFURL returned nil error for closed metadata")
	}
}
