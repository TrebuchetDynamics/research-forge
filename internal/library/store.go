package library

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Store persists PaperRecords for local library workflows.
type Store struct {
	path string
}

// OpenStore opens a local JSON-backed PaperRecord store.
func OpenStore(path string) (Store, error) {
	if strings.TrimSpace(path) == "" {
		return Store{}, fmt.Errorf("library store path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Store{}, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte("[]\n"), 0o644); err != nil {
			return Store{}, err
		}
	} else if err != nil {
		return Store{}, err
	}
	return Store{path: path}, nil
}

// Create inserts a new PaperRecord.
func (s Store) Create(record PaperRecord) error {
	records, err := s.List()
	if err != nil {
		return err
	}
	key := recordKey(record)
	if key == "" {
		return fmt.Errorf("paper record identifier is required")
	}
	for _, existing := range records {
		if recordKey(existing) == key {
			return fmt.Errorf("paper record already exists")
		}
	}
	records = append(records, record)
	return s.write(records)
}

// Update replaces an existing PaperRecord by identifier.
func (s Store) Update(record PaperRecord) error {
	records, err := s.List()
	if err != nil {
		return err
	}
	key := recordKey(record)
	for i, existing := range records {
		if recordKey(existing) == key {
			records[i] = record
			return s.write(records)
		}
	}
	return fmt.Errorf("paper record not found")
}

// ReplaceAll atomically replaces the store contents with the provided records.
func (s Store) ReplaceAll(records []PaperRecord) error {
	return s.write(records)
}

// List returns all PaperRecords sorted by title.
func (s Store) List() ([]PaperRecord, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	var records []PaperRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	sort.Slice(records, func(i, j int) bool { return records[i].Title < records[j].Title })
	return records, nil
}

// Search returns PaperRecords whose title or abstract contains query.
func (s Store) Search(query string) ([]PaperRecord, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, fmt.Errorf("library search query is required")
	}
	records, err := s.List()
	if err != nil {
		return nil, err
	}
	results := []PaperRecord{}
	for _, record := range records {
		text := strings.ToLower(record.Title + "\n" + record.Abstract)
		if strings.Contains(text, query) {
			results = append(results, record)
		}
	}
	return results, nil
}

func (s Store) write(records []PaperRecord) error {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.path, data, 0o644)
}

func recordKey(record PaperRecord) string {
	ids := record.Identifiers
	switch {
	case ids.DOI != "":
		return "doi:" + ids.DOI
	case ids.OpenAlexID != "":
		return "openalex:" + ids.OpenAlexID
	case ids.ArXivID != "":
		return "arxiv:" + ids.ArXivID
	case ids.PMID != "":
		return "pmid:" + ids.PMID
	case ids.CrossrefID != "":
		return "crossref:" + ids.CrossrefID
	case ids.SemanticScholarID != "":
		return "s2:" + ids.SemanticScholarID
	default:
		return ""
	}
}
