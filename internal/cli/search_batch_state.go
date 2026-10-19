package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

// searchBatchStateFile is the crash-safety checkpoint for `search batch`. It is
// rewritten atomically after every attempted (query, source) pair so that an
// interrupted sweep — harness timeout, SIGTERM, SIGKILL, or a network hang —
// leaves a durable record of exactly which pairs completed, which failed, and
// which were never attempted. A later `search batch --resume` continues from
// this file instead of restarting the sweep or orphaning results in a temp
// staging directory.
const searchBatchStateFile = "batch-state.json"

// searchBatchStatePair records one completed (query, source) search pair.
type searchBatchStatePair struct {
	Source  string `json:"source"`
	Query   string `json:"query"`
	RawFile string `json:"rawFile"`
	Records int    `json:"records"`
}

// searchBatchState is the on-disk checkpoint written incrementally during a
// batch sweep and read by `search batch --resume`.
type searchBatchState struct {
	SchemaVersion string                 `json:"schemaVersion"`
	CreatedAt     string                 `json:"createdAt"`
	UpdatedAt     string                 `json:"updatedAt"`
	Queries       []string               `json:"queries"`
	Sources       []string               `json:"sources"`
	Limit         int                    `json:"limit"`
	Completed     []searchBatchStatePair `json:"completed,omitempty"`
	Failures      []searchBatchFailure   `json:"failures,omitempty"`
	Interrupted   bool                   `json:"interrupted,omitempty"`
}

// searchBatchPairKey uniquely identifies a (source, query) sweep pair.
func searchBatchPairKey(source, query string) string {
	return source + "\x00" + query
}

// hasCompleted reports whether the pair already completed in a prior (or the
// current) sweep.
func (state *searchBatchState) hasCompleted(source, query string) bool {
	key := searchBatchPairKey(source, query)
	for _, pair := range state.Completed {
		if searchBatchPairKey(pair.Source, pair.Query) == key {
			return true
		}
	}
	return false
}

// recordCompleted appends (or refreshes) a completed pair entry.
func (state *searchBatchState) recordCompleted(pair searchBatchStatePair) {
	key := searchBatchPairKey(pair.Source, pair.Query)
	for i, existing := range state.Completed {
		if searchBatchPairKey(existing.Source, existing.Query) == key {
			state.Completed[i] = pair
			return
		}
	}
	state.Completed = append(state.Completed, pair)
}

// recordFailure appends (or refreshes) a failure entry for a pair.
func (state *searchBatchState) recordFailure(failure searchBatchFailure) {
	key := searchBatchPairKey(failure.Source, failure.Query)
	for i, existing := range state.Failures {
		if searchBatchPairKey(existing.Source, existing.Query) == key {
			state.Failures[i] = failure
			return
		}
	}
	state.Failures = append(state.Failures, failure)
}

// clearFailure drops a previously recorded failure, e.g. when a resumed retry
// succeeds.
func (state *searchBatchState) clearFailure(source, query string) {
	key := searchBatchPairKey(source, query)
	kept := state.Failures[:0]
	for _, failure := range state.Failures {
		if searchBatchPairKey(failure.Source, failure.Query) == key {
			continue
		}
		kept = append(kept, failure)
	}
	state.Failures = kept
}

// readSearchBatchState loads the checkpoint from a topic directory. A missing
// file is not an error (fresh sweep); a corrupt file is, so a resume never
// silently restarts a sweep whose prior work cannot be trusted.
func readSearchBatchState(dir string) (searchBatchState, bool, error) {
	path := filepath.Join(dir, searchBatchStateFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return searchBatchState{}, false, nil
		}
		return searchBatchState{}, false, err
	}
	var state searchBatchState
	if err := json.Unmarshal(data, &state); err != nil {
		return searchBatchState{}, true, fmt.Errorf("parse %s: %w", path, err)
	}
	return state, true, nil
}

// writeSearchBatchState atomically rewrites the checkpoint in a topic dir.
func writeSearchBatchState(dir string, state searchBatchState) error {
	return writeJSONFile(filepath.Join(dir, searchBatchStateFile), state)
}

// loadSearchBatchResults reads an accumulated results.jsonl back into paper
// records so a resumed run can regenerate aggregate artifacts from the union
// of prior and newly fetched results.
func loadSearchBatchResults(path string) ([]library.PaperRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	papers := []library.PaperRecord{}
	line := 0
	for _, raw := range splitJSONLines(data) {
		line++
		var paper library.PaperRecord
		if err := json.Unmarshal(raw, &paper); err != nil {
			return nil, fmt.Errorf("parse %s line %d: %w", path, line, err)
		}
		papers = append(papers, paper)
	}
	return papers, nil
}

// splitJSONLines splits a JSONL byte slice into non-empty line payloads.
func splitJSONLines(data []byte) [][]byte {
	lines := [][]byte{}
	start := 0
	for i, b := range data {
		if b == '\n' {
			if chunk := trimSpaceBytes(data[start:i]); len(chunk) > 0 {
				lines = append(lines, chunk)
			}
			start = i + 1
		}
	}
	if chunk := trimSpaceBytes(data[start:]); len(chunk) > 0 {
		lines = append(lines, chunk)
	}
	return lines
}

func trimSpaceBytes(data []byte) []byte {
	start, end := 0, len(data)
	for start < end && isJSONLSpace(data[start]) {
		start++
	}
	for end > start && isJSONLSpace(data[end-1]) {
		end--
	}
	return data[start:end]
}

func isJSONLSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r'
}

// searchBatchPendingPairs counts sweep pairs that are neither completed,
// failed, nor on a source skipped for missing configuration, given the current
// query list and per-source configuration errors.
func searchBatchPendingPairs(state *searchBatchState, queries, sources []string, configurationErrors map[string]string) int {
	pending := 0
	for _, query := range queries {
		for _, source := range sources {
			if configurationErrors[source] != "" {
				continue
			}
			if state.hasCompleted(source, query) {
				continue
			}
			if searchBatchStateHasFailure(state, source, query) {
				continue
			}
			pending++
		}
	}
	return pending
}

func searchBatchStateHasFailure(state *searchBatchState, source, query string) bool {
	key := searchBatchPairKey(source, query)
	for _, failure := range state.Failures {
		if searchBatchPairKey(failure.Source, failure.Query) == key {
			return true
		}
	}
	return false
}

// ensureSearchBatchStateFileRegistered keeps the checkpoint inside the staged
// output transaction so a graceful failure removes or restores it alongside
// the other batch artifacts.
func ensureSearchBatchStateFileRegistered(relativePaths []string) []string {
	for _, path := range relativePaths {
		if path == searchBatchStateFile {
			return relativePaths
		}
	}
	return append(relativePaths, searchBatchStateFile)
}
