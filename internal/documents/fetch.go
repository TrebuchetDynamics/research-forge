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

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
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
	dir := filepath.Join(projectPath, "documents", "arxiv")
	path := filepath.Join(dir, safeDocumentName(arxivID)+ext)
	if err := validateDocumentOutputPath(projectPath, path); err != nil {
		return DocumentAsset{}, err
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
	if err := ensureDocumentsGitignored(projectPath); err != nil {
		return DocumentAsset{}, err
	}
	if err := writeDocumentFile(projectPath, path, data, 0o644); err != nil {
		return DocumentAsset{}, err
	}
	return NewDocumentAsset(DocumentAssetInput{PaperID: arxivID, AcquisitionSource: "arxiv-" + kind, License: "arXiv", OAStatus: "green", LocalPath: path, MIMEType: mimeType, LocalOnly: localOnly})
}

// FetchPDFByDOI downloads a legal open-access PDF into project storage.
func FetchPDFByDOI(ctx context.Context, projectPath, doi string, metadata OpenAccessMetadata) (DocumentAsset, error) {
	return FetchPDF(ctx, projectPath, doi, metadata)
}

// FetchPDF downloads a legal open-access PDF into project storage.
func FetchPDF(ctx context.Context, projectPath, paperID string, metadata OpenAccessMetadata) (DocumentAsset, error) {
	paperID = strings.TrimSpace(paperID)
	if paperID == "" {
		return DocumentAsset{}, fmt.Errorf("paper id is required")
	}
	pdfURL, err := SelectLegalPDFURL(metadata)
	if err != nil {
		return DocumentAsset{}, err
	}
	if err := validatePDFURL(pdfURL); err != nil {
		return DocumentAsset{}, err
	}
	dir := filepath.Join(projectPath, "documents", "open-access")
	path := filepath.Join(dir, safeDocumentName(paperID)+".pdf")
	if err := validateDocumentOutputPath(projectPath, path); err != nil {
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
	if err := ensureDocumentsGitignored(projectPath); err != nil {
		return DocumentAsset{}, err
	}
	if err := writeDocumentFile(projectPath, path, data, 0o644); err != nil {
		return DocumentAsset{}, err
	}
	return NewDocumentAsset(DocumentAssetInput{
		PaperID:           paperID,
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

func ensureDocumentsGitignored(projectPath string) error {
	path := filepath.Join(projectPath, ".gitignore")
	if err := validateDocumentOutputPath(projectPath, path); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return writeDocumentFile(projectPath, path, []byte("documents/\n"), 0o644)
	}
	if err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		switch strings.TrimSpace(line) {
		case "documents/", "/documents/", "documents/**":
			return nil
		}
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	data = append(data, []byte("documents/\n")...)
	return writeDocumentFile(projectPath, path, data, info.Mode().Perm())
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

func validateDocumentOutputPath(projectPath, path string) error {
	root, err := filepath.Abs(projectPath)
	if err != nil {
		return err
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("document output escapes project: %s", path)
	}
	if info, err := os.Lstat(target); err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("document output is not a regular file: %s", path)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	for dir := filepath.Dir(target); dir != root; dir = filepath.Dir(dir) {
		info, err := os.Lstat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("document output parent is not a directory: %s", dir)
		}
	}
	return nil
}

func writeDocumentFile(projectPath, path string, data []byte, mode os.FileMode) error {
	if err := validateDocumentOutputPath(projectPath, path); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := validateDocumentOutputPath(projectPath, path); err != nil {
		return err
	}
	return filetxn.Replace(path, data, mode)
}
