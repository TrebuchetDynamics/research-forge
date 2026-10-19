package cli

import (
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/research-forge/internal/search"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

// Compatibility names for the CLI artifact adapters; state and resume belong to search.
const searchBatchStateFile = search.BatchStateFile

type searchBatchState = search.BatchState
type searchBatchStatePair = search.BatchPair

var readSearchBatchState = search.ReadBatchState
var writeSearchBatchState = search.WriteBatchState
var loadSearchBatchResults = search.LoadBatchResults
var searchBatchPendingPairs = search.PendingPairs

// preserveSearchBatchUsage restores a prior artifact prefix without refunding
// network work. Failed fresh runs still leave no checkpoint, for CLI parity.
func preserveSearchBatchUsage(dir string, prior, current search.BatchState) error {
	prior.RequestsUsed = current.RequestsUsed
	prior.Budget = current.Budget
	prior.StopReason = sources.StopErrorThreshold
	prior.UpdatedAt = current.UpdatedAt
	if err := search.WriteBatchState(dir, prior); err != nil {
		return err
	}
	var manifest searchBatchManifest
	path := filepath.Join(dir, "manifest.json")
	if err := readJSONFile(path, &manifest); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if manifest.SchemaVersion != "1" {
		return nil
	}
	manifest.BatchProgress = prior.BatchProgress
	return writeJSONFile(path, manifest)
}
