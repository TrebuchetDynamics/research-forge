package claims

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

var (
	ErrLibraryRead = errors.New("claim library read failed")
	ErrParsedRead  = errors.New("claim parsed documents read failed")
)

// View adds computed attribution, never a truth judgment or a caller-set level.
type View struct {
	Claim
	SemanticSupportLevel string `json:"semanticSupportLevel,omitempty"`
}

func ViewClaim(record Claim) View {
	view := View{Claim: record}
	if record.Facets.SemanticSupport != nil {
		view.SemanticSupportLevel = "agent_reported"
	}
	return view
}

type ProjectCheckOptions struct {
	Project   string
	ID        string
	Timestamp time.Time
	// Only agent-reported semantic support can be submitted by a host.
	// Human approval is preserved from storage, never set by this operation.
	SemanticSupport *SemanticSupport
}

type ProjectCheckResult struct {
	Claim    View            `json:"claim"`
	Evidence []EvidenceCheck `json:"evidence"`
}

// CheckProject checks stored offline evidence and appends one complete snapshot.
// CLI and MCP share this operation, including persistence and attribution.
func CheckProject(opts ProjectCheckOptions) (ProjectCheckResult, error) {
	if strings.TrimSpace(opts.Project) == "" || strings.TrimSpace(opts.ID) == "" {
		return ProjectCheckResult{}, fmt.Errorf("project and claim ID are required")
	}
	record, err := Show(opts.Project, opts.ID)
	if err != nil {
		return ProjectCheckResult{}, err
	}
	records, err := projectLibraryRecords(opts.Project)
	if err != nil {
		return ProjectCheckResult{}, fmt.Errorf("%w: %w", ErrLibraryRead, err)
	}
	docs, err := projectParsedDocuments(opts.Project)
	if err != nil {
		return ProjectCheckResult{}, fmt.Errorf("%w: %w", ErrParsedRead, err)
	}
	if opts.SemanticSupport != nil {
		record.Facets.SemanticSupport = opts.SemanticSupport
	}
	result, err := Check(record, CheckInput{LibraryRecords: records, ParsedDocuments: docs, Timestamp: opts.Timestamp})
	if err != nil {
		return ProjectCheckResult{}, err
	}
	if err := Append(opts.Project, result.Claim); err != nil {
		return ProjectCheckResult{}, err
	}
	return ProjectCheckResult{Claim: ViewClaim(result.Claim), Evidence: result.Evidence}, nil
}

func projectParsedDocuments(project string) ([]parsing.ParsedDocument, error) {
	dir := filepath.Join(project, "parsed")
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return []parsing.ParsedDocument{}, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("parsed document path is not a directory: %s", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	docs := []parsing.ParsedDocument{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var doc parsing.ParsedDocument
		// A file that disappears after listing is an error, not missing evidence.
		if err := readStoredJSON(filepath.Join(dir, entry.Name()), &doc); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func projectLibraryRecords(project string) ([]library.PaperRecord, error) {
	path := filepath.Join(project, "data", "library.json")
	if _, err := os.Lstat(path); err != nil {
		if os.IsNotExist(err) {
			return []library.PaperRecord{}, nil
		}
		return nil, err
	}
	store, err := library.OpenStore(path)
	if err != nil {
		return nil, err
	}
	return store.List()
}

func readStoredJSON(path string, target any) error {
	file, err := filetxn.OpenRegular(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("stored evidence %s must contain exactly one JSON value", path)
	}
	return nil
}
