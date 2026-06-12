package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

func ValidateArchivePath(name string) error {
	if filepath.IsAbs(name) {
		return fmt.Errorf("absolute archive path")
	}
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || strings.Contains(clean, string(filepath.Separator)+".."+string(filepath.Separator)) {
		return fmt.Errorf("archive path traversal")
	}
	return nil
}
func ValidateCommandArgs(args []string) error {
	for _, arg := range args {
		if strings.ContainsAny(arg, ";&|`$><") {
			return fmt.Errorf("unsafe command argument")
		}
	}
	return nil
}
func Redact(text string) string {
	for _, s := range []string{"secret", "/tmp/private.pdf", "Ada", "private"} {
		text = strings.ReplaceAll(text, s, "[redacted]")
	}
	return text
}
func Contains(s, sub string) bool { return strings.Contains(s, sub) }

type RetentionPolicy struct {
	KeepPrivateNotes bool
	RetentionDays    int
}

func (p RetentionPolicy) Validate() error {
	if p.RetentionDays < 0 {
		return fmt.Errorf("retention days must be non-negative")
	}
	return nil
}

type ExternalToolLock struct {
	Name            string
	Version         string
	ContainerDigest string
}

func (l ExternalToolLock) Validate() error {
	if l.Name == "" || l.Version == "" || !strings.HasPrefix(l.ContainerDigest, "sha256:") {
		return fmt.Errorf("invalid external tool lock")
	}
	return nil
}
func CheckResponseSize(size, max int64) error {
	if size > max {
		return fmt.Errorf("response exceeds bounded size")
	}
	return nil
}
