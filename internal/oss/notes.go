package oss

import (
	"fmt"
	"path/filepath"
	"strings"
)

// WriteStudyNote writes a safe study-note template for an OSS repository and area.
func WriteStudyNote(projectPath, name, area string) (string, error) {
	study, err := NewRepositoryStudy(RepositoryStudyInput{Name: name})
	if err != nil {
		return "", err
	}
	area = strings.TrimSpace(area)
	if area == "" {
		area = "general"
	}
	filename := safeNoteName(area) + ".md"
	dir := filepath.Join(projectPath, "opensource", "notes", study.Owner, study.Repo)
	path := filepath.Join(dir, filename)
	content := fmt.Sprintf(`# %s — %s

## Summary

- What is this repository useful for in this research workflow?

## License and provenance

- Detected license:
- Source URL:
- Review notes:

## Architecture observations

- Components:
- Data flow:
- Reusable ideas:

## Safety

Do not copy external source code into ResearchForge production code without explicit license and provenance review.
`, study.Name, area)
	return path, writeOutput(path, []byte(content))
}

func safeNoteName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	if len(parts) == 0 {
		return "general"
	}
	return strings.Join(parts, "-")
}
