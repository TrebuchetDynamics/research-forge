package sources

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

// ResponseCache stores raw source payloads for reproducible connector runs.
type ResponseCache struct {
	root string
}

// NewResponseCache creates a filesystem-backed source response cache.
func NewResponseCache(root string) ResponseCache {
	return ResponseCache{root: root}
}

// Write stores a raw source response and returns a stable cache reference.
func (c ResponseCache) Write(source, query string, payload []byte) (string, error) {
	if strings.TrimSpace(source) == "" {
		return "", fmt.Errorf("source is required")
	}
	if strings.TrimSpace(query) == "" {
		return "", fmt.Errorf("query is required")
	}
	if c.root == "" {
		return "", fmt.Errorf("cache root is required")
	}
	hash := sha256.Sum256([]byte(source + "\x00" + query))
	ref := filepath.ToSlash(filepath.Join(slug(source), hex.EncodeToString(hash[:])[:16]+".json"))
	path := filepath.Join(c.root, filepath.FromSlash(ref))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	mode := os.FileMode(0o644)
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("cache entry is not a regular file: %s", path)
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := filetxn.Replace(path, payload, mode); err != nil {
		return "", err
	}
	return ref, nil
}

// Read returns a raw source response by cache reference.
func (c ResponseCache) Read(ref string) ([]byte, error) {
	if c.root == "" {
		return nil, fmt.Errorf("cache root is required")
	}
	if ref == "" || strings.Contains(ref, "..") || filepath.IsAbs(ref) {
		return nil, fmt.Errorf("invalid cache ref")
	}
	path := filepath.Join(c.root, filepath.FromSlash(ref))
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("cache entry is not a regular file: %s", path)
	}
	return os.ReadFile(path)
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
