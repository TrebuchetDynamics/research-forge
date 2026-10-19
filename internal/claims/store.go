package claims

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

// RelativePath is the append-only project claim ledger. Each line is a complete
// Claim snapshot; a later snapshot with the same ID supersedes, never edits, it.
const RelativePath = "data/claims.jsonl"

var ErrNotFound = errors.New("claim not found")

// Append records a new snapshot without changing any existing ledger bytes.
// Facets refer to this exact text/evidence, so callers must not carry assessments
// across edits without reassessing them. Nothing here performs a facet check.
func Append(projectPath string, record Claim) error {
	if strings.TrimSpace(projectPath) == "" {
		return fmt.Errorf("claim store requires a project path")
	}
	if err := validateClaim(record); err != nil {
		return err
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	// Refuse to append to a corrupt or incomplete ledger, rather than masking it
	// or joining the next JSON object onto an unterminated record.
	if _, err := List(projectPath); err != nil {
		return err
	}
	path := filepath.Join(projectPath, RelativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return filetxn.Append(path, append(data, '\n'), 0o644)
}

// List returns the latest snapshot per ID in first-seen order. A missing ledger
// is empty; malformed or incomplete records are errors, not an empty result.
func List(projectPath string) ([]Claim, error) {
	if strings.TrimSpace(projectPath) == "" {
		return nil, fmt.Errorf("claim store requires a project path")
	}
	file, err := filetxn.OpenRegular(filepath.Join(projectPath, RelativePath))
	if os.IsNotExist(err) {
		return []Claim{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	records := []Claim{}
	positions := map[string]int{}
	reader := bufio.NewReader(file)
	for line := 1; ; line++ {
		data, err := reader.ReadBytes('\n')
		if err == io.EOF && len(data) == 0 {
			return records, nil
		}
		if err != nil {
			return nil, fmt.Errorf("claim ledger line %d: incomplete record: %w", line, err)
		}
		var record Claim
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, fmt.Errorf("claim ledger line %d: %w", line, err)
		}
		if err := validateClaim(record); err != nil {
			return nil, fmt.Errorf("claim ledger line %d: %w", line, err)
		}
		if index, found := positions[record.ID]; found {
			records[index] = record
		} else {
			positions[record.ID] = len(records)
			records = append(records, record)
		}
	}
}

// Show returns the latest snapshot of id, or ErrNotFound.
func Show(projectPath, id string) (Claim, error) {
	records, err := List(projectPath)
	if err != nil {
		return Claim{}, err
	}
	for _, record := range records {
		if record.ID == id {
			return record, nil
		}
	}
	return Claim{}, fmt.Errorf("%w: %s", ErrNotFound, id)
}

func validateClaim(record Claim) error {
	if record.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported claim schema version %q", record.SchemaVersion)
	}
	if strings.TrimSpace(record.ID) == "" || strings.TrimSpace(record.Text) == "" {
		return fmt.Errorf("claim requires an ID and text")
	}
	for i, reference := range record.Evidence {
		if strings.TrimSpace(reference.Identifier.Scheme) == "" || strings.TrimSpace(reference.Identifier.Value) == "" {
			return fmt.Errorf("evidence %d requires a source scheme and value", i)
		}
		switch reference.Relation {
		case RelationSupports, RelationContradicts, RelationContext:
		default:
			return fmt.Errorf("evidence %d has unsupported relation %q", i, reference.Relation)
		}
		if hash := reference.ContentHash; hash != "" {
			digest, err := hex.DecodeString(strings.TrimPrefix(hash, "sha256:"))
			if !strings.HasPrefix(hash, "sha256:") || err != nil || len(digest) != 32 || hash != strings.ToLower(hash) {
				return fmt.Errorf("evidence %d requires a sha256:<lowercase hex> content hash", i)
			}
		}
		if offsets := reference.Offsets; offsets != nil {
			if offsets.Start < 0 || offsets.End < offsets.Start || strings.TrimSpace(offsets.ParserName) == "" || strings.TrimSpace(offsets.ParserVersion) == "" {
				return fmt.Errorf("evidence %d requires ordered nonnegative offsets and parser name/version", i)
			}
		}
	}
	for _, facet := range []struct {
		name  string
		value *Assessment
	}{
		{"source identity", record.Facets.SourceIdentity},
		{"content integrity", record.Facets.ContentIntegrity},
		{"passage match", record.Facets.PassageMatch},
	} {
		if facet.value == nil {
			continue
		}
		switch facet.value.Checker.Kind {
		case CheckerEngine, CheckerAgent, CheckerHuman:
		default:
			return fmt.Errorf("%s requires a known checker kind", facet.name)
		}
		if err := validateAssessment(*facet.value, facet.value.Checker.Kind); err != nil {
			return fmt.Errorf("%s: %w", facet.name, err)
		}
	}
	return nil
}
