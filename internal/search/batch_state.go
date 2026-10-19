package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

const BatchStateFile = "batch-state.json"

type BatchPair struct {
	Source  string `json:"source"`
	Query   string `json:"query"`
	RawFile string `json:"rawFile"`
	Records int    `json:"records"`
}

type BatchFailure struct {
	Source  string `json:"source"`
	Query   string `json:"query"`
	Error   string `json:"error"`
	Skipped bool   `json:"skipped,omitempty"`
}

// BatchState is the durable checkpoint shared by CLI and service consumers.
// RequestsUsed is persisted before dispatch, so interrupted calls consume budget.
type BatchState struct {
	JobID         string         `json:"job_id,omitempty"`
	SchemaVersion string         `json:"schemaVersion"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
	Queries       []string       `json:"queries"`
	Sources       []string       `json:"sources"`
	Limit         int            `json:"limit"`
	Completed     []BatchPair    `json:"completed,omitempty"`
	Failures      []BatchFailure `json:"failures,omitempty"`
	Interrupted   bool           `json:"interrupted,omitempty"`
	// ResultCount bounds the committed prefix of results.jsonl. A nil value
	// identifies legacy checkpoints, whose entire result file is retained.
	ResultCount *int `json:"result_count,omitempty"`
	BatchProgress
}

func PairKey(source, query string) string { return source + "\x00" + query }
func (s *BatchState) HasCompleted(source, query string) bool {
	for _, p := range s.Completed {
		if PairKey(p.Source, p.Query) == PairKey(source, query) {
			return true
		}
	}
	return false
}
func (s *BatchState) RecordCompleted(p BatchPair) {
	for i, old := range s.Completed {
		if PairKey(old.Source, old.Query) == PairKey(p.Source, p.Query) {
			s.Completed[i] = p
			return
		}
	}
	s.Completed = append(s.Completed, p)
}
func (s *BatchState) RecordFailure(f BatchFailure) {
	for i, old := range s.Failures {
		if PairKey(old.Source, old.Query) == PairKey(f.Source, f.Query) {
			s.Failures[i] = f
			return
		}
	}
	s.Failures = append(s.Failures, f)
}
func (s *BatchState) ClearFailure(source, query string) {
	kept := s.Failures[:0]
	for _, f := range s.Failures {
		if PairKey(f.Source, f.Query) != PairKey(source, query) {
			kept = append(kept, f)
		}
	}
	s.Failures = kept
}

// Resume retains cumulative usage, coverage and unattempted failures. Old
// checkpoints infer a minimum usage from their attempted pairs.
func (s *BatchState) Resume(queries, sources []string, limit int) {
	if s.RequestsUsed == 0 {
		s.RequestsUsed = len(s.Completed)
		for _, f := range s.Failures {
			if !f.Skipped {
				s.RequestsUsed++
			}
		}
	}
	s.Queries, s.Sources, s.Limit = queries, sources, limit
	s.Interrupted = false
	kept := []BatchFailure{}
	for _, f := range s.Failures {
		if f.Query != "" {
			kept = append(kept, f)
		}
	}
	s.Failures = kept
	s.ConsecutiveErrors = 0
}

func ReadBatchState(dir string) (BatchState, bool, error) {
	path := filepath.Join(dir, BatchStateFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return BatchState{}, false, nil
	}
	if err != nil {
		return BatchState{}, false, err
	}
	var state BatchState
	if err := json.Unmarshal(data, &state); err != nil {
		return BatchState{}, true, fmt.Errorf("parse %s: %w", path, err)
	}
	return state, true, nil
}
func WriteBatchState(dir string, state BatchState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return filetxn.Replace(filepath.Join(dir, BatchStateFile), append(data, '\n'), 0o644)
}
func LoadBatchResults(path string) ([]library.PaperRecord, error) {
	return LoadCheckpointResults(path, nil)
}

// LoadCheckpointResults ignores any uncommitted (even torn) trailing append.
func LoadCheckpointResults(path string, count *int) ([]library.PaperRecord, error) {
	if count != nil && *count < 0 {
		return nil, fmt.Errorf("negative result_count in checkpoint")
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) && (count == nil || *count == 0) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	papers := []library.PaperRecord{}
	for i, line := range bytes.Split(data, []byte("\n")) {
		if count != nil && len(papers) >= *count {
			break
		}
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var paper library.PaperRecord
		if err := json.Unmarshal(line, &paper); err != nil {
			return nil, fmt.Errorf("parse %s line %d: %w", path, i+1, err)
		}
		papers = append(papers, paper)
	}
	if count != nil && len(papers) != *count {
		return nil, fmt.Errorf("%s has %d records; checkpoint requires %d", path, len(papers), *count)
	}
	return papers, nil
}
func PendingPairs(state *BatchState, queries, sources []string, configurationErrors map[string]string) int {
	pending := 0
	for _, q := range queries {
		for _, src := range sources {
			if configurationErrors[src] != "" || state.HasCompleted(src, q) {
				continue
			}
			failed := false
			for _, f := range state.Failures {
				if PairKey(f.Source, f.Query) == PairKey(src, q) {
					failed = true
					break
				}
			}
			if !failed {
				pending++
			}
		}
	}
	return pending
}
