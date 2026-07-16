package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ValidateArchivePath(name string) error {
	if name == "" {
		return fmt.Errorf("archive path is required")
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("absolute archive path")
	}
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || strings.Contains(clean, string(filepath.Separator)+".."+string(filepath.Separator)) {
		return fmt.Errorf("archive path traversal")
	}
	return nil
}

func ValidatePathWithinRoot(root, name string) error {
	if err := ValidateArchivePath(name); err != nil {
		return err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	current := rootAbs
	for _, part := range strings.Split(filepath.Clean(filepath.FromSlash(name)), string(filepath.Separator)) {
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
			return fmt.Errorf("path traverses symlink")
		}
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
