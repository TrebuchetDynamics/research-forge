package documents

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxPDFDownloadBytes int64 = 100 << 20

var pdfHTTPClient = &http.Client{Timeout: 30 * time.Second}

// FetchArXivAsset downloads an arXiv PDF or TeX source asset into project storage.
func FetchArXivAsset(ctx context.Context, projectPath, arxivID, assetURL, kind string) (DocumentAsset, error) {
	arxivID = strings.TrimSpace(arxivID)
	if arxivID == "" {
		return DocumentAsset{}, fmt.Errorf("arxiv id is required")
	}
	if err := validatePDFURL(assetURL); err != nil {
		return DocumentAsset{}, err
	}
	mimeType := "application/pdf"
	ext := ".pdf"
	localOnly := false
	if kind == "source" {
		mimeType = "application/gzip"
		ext = ".tar.gz"
		localOnly = true
	} else if kind != "pdf" {
		return DocumentAsset{}, fmt.Errorf("arxiv asset kind must be pdf or source")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return DocumentAsset{}, err
	}
	response, err := pdfHTTPClient.Do(request)
	if err != nil {
		return DocumentAsset{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return DocumentAsset{}, fmt.Errorf("arxiv fetch status %d", response.StatusCode)
	}
	data, err := readPDFResponse(response)
	if err != nil {
		return DocumentAsset{}, err
	}
	dir := filepath.Join(projectPath, "documents", "arxiv")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return DocumentAsset{}, err
	}
	path := filepath.Join(dir, safeDocumentName(arxivID)+ext)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return DocumentAsset{}, err
	}
	return NewDocumentAsset(DocumentAssetInput{PaperID: arxivID, AcquisitionSource: "arxiv-" + kind, License: "arXiv", OAStatus: "green", LocalPath: path, MIMEType: mimeType, LocalOnly: localOnly})
}

// FetchPDFByDOI downloads a legal open-access PDF into project storage.
func FetchPDFByDOI(ctx context.Context, projectPath, doi string, metadata OpenAccessMetadata) (DocumentAsset, error) {
	pdfURL, err := SelectLegalPDFURL(metadata)
	if err != nil {
		return DocumentAsset{}, err
	}
	if err := validatePDFURL(pdfURL); err != nil {
		return DocumentAsset{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, pdfURL, nil)
	if err != nil {
		return DocumentAsset{}, err
	}
	response, err := pdfHTTPClient.Do(request)
	if err != nil {
		return DocumentAsset{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return DocumentAsset{}, fmt.Errorf("pdf fetch status %d", response.StatusCode)
	}
	data, err := readPDFResponse(response)
	if err != nil {
		return DocumentAsset{}, err
	}
	dir := filepath.Join(projectPath, "documents", "open-access")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return DocumentAsset{}, err
	}
	path := filepath.Join(dir, safeDocumentName(doi)+".pdf")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return DocumentAsset{}, err
	}
	return NewDocumentAsset(DocumentAssetInput{
		PaperID:           doi,
		AcquisitionSource: "open-access-pdf",
		License:           metadata.License,
		OAStatus:          metadata.OAStatus,
		LocalPath:         path,
		MIMEType:          "application/pdf",
	})
}

func validatePDFURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("pdf URL must be absolute http(s)")
	}
	return nil
}

func readPDFResponse(response *http.Response) ([]byte, error) {
	if response.ContentLength > maxPDFDownloadBytes {
		return nil, fmt.Errorf("pdf response too large: %d bytes exceeds %d", response.ContentLength, maxPDFDownloadBytes)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxPDFDownloadBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxPDFDownloadBytes {
		return nil, fmt.Errorf("pdf response too large: exceeds %d", maxPDFDownloadBytes)
	}
	return data, nil
}

func safeDocumentName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	if len(parts) == 0 {
		return "document"
	}
	return strings.Join(parts, "-")
}
