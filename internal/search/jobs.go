package search

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/TrebuchetDynamics/research-forge/internal/jobs"
)

// BatchJobKind identifies a persistent search sweep for every host surface.
const BatchJobKind = "search.batch"

// BatchJobProgress counts committed pairs in the active worklist, not stale
// pairs retained from an earlier search with different queries or sources.
func BatchJobProgress(state *BatchState) jobs.Progress {
	progress := jobs.Progress{Total: len(state.Queries) * len(state.Sources)}
	for _, query := range state.Queries {
		for _, source := range state.Sources {
			if state.HasCompleted(source, query) {
				progress.Completed++
			}
		}
	}
	return progress
}

// BatchJobInput saves resolved inputs rather than references to mutable query
// files. Credentials and connector endpoints remain in host configuration.
// The artifact options are shared with adapters so resumption also republishes
// the original outputs and project import, not just the network worklist.
type BatchJobInput struct {
	// ResumeFrom identifies the checkpoint deliberately inherited at creation.
	// Nil means a fresh job even if older artifacts still exist. A pointer to
	// an empty ID denotes a legacy checkpoint that predates persistent jobs.
	ResumeFrom      *string  `json:"resume_from,omitempty"`
	Queries         []string `json:"queries"`
	Sources         []string `json:"sources"`
	Limit           int      `json:"limit"`
	Depth           string   `json:"depth,omitempty"`
	MaxRequests     int      `json:"max_requests,omitempty"`
	ContinueOnError bool     `json:"continue_on_error"`
	WriteStats      bool     `json:"write_stats"`
	FetchPDFs       bool     `json:"fetch_pdfs"`
	Project         string   `json:"project,omitempty"`
}

// RunBatchJob owns registration, lease and resume identity for every host.
// With an ID it restores saved inputs; without one it starts a request (or
// extends a checkpoint if resume is true). The adapter's handler publishes the
// artifacts and calls RunBatch with the supplied lease before returning.
func RunBatchJob(ctx context.Context, dir, id string, input BatchJobInput, resume bool, handler func(context.Context, *jobs.Run, BatchJobInput, bool) error) (jobs.Job, error) {
	store := jobs.NewStore(dir)
	explicitResume := id != ""
	if resume && !explicitResume {
		state, exists, err := ReadBatchState(dir)
		if err != nil {
			return jobs.Job{}, err
		}
		if exists {
			input.ResumeFrom = &state.JobID
			if state.JobID != "" {
				job, err := store.Status(state.JobID)
				if err != nil {
					return jobs.Job{}, err
				}
				// Extending an already completed worklist is a new job.
				if job.Status != jobs.Completed {
					id = job.ID
				}
			}
		}
	}
	if id == "" {
		job, err := store.Create(BatchJobKind, input)
		if err != nil {
			return jobs.Job{}, err
		}
		id = job.ID
	}
	return store.Run(ctx, id, func(ctx context.Context, run *jobs.Run) error {
		if run.Job().Kind != BatchJobKind {
			return fmt.Errorf("unknown batch job kind %q", run.Job().Kind)
		}
		if explicitResume {
			input = BatchJobInput{}
			if err := json.Unmarshal(run.Job().Input, &input); err != nil {
				return err
			}
			// Registration alone cannot supersede a writer: another host may
			// register, then fail to acquire the execution lease. A newer job
			// that did start can already have replaced results even if it died
			// before publishing its first batch checkpoint.
			all, err := store.List()
			if err != nil {
				return err
			}
			for _, other := range all {
				if other.CreatedAt.After(run.Job().CreatedAt) && other.Status != jobs.Queued {
					return fmt.Errorf("job %s was superseded by a newer job", id)
				}
			}
			resume = true
		}
		if resume {
			// Revalidate implicit as well as explicit checkpoint inheritance
			// while holding the writer lease, not only during registration.
			state, exists, err := ReadBatchState(dir)
			if err != nil {
				return err
			}
			if !exists || state.JobID != id {
				// Registration precedes the first checkpoint. Recover without
				// inheriting unrelated artifacts from an older search.
				resume = input.ResumeFrom != nil
				if resume && (!exists || state.JobID != *input.ResumeFrom) {
					return fmt.Errorf("job %s inherited checkpoint %s which is no longer available", id, *input.ResumeFrom)
				}
			}
		}
		if err := run.SetInput(input); err != nil {
			return err
		}
		return handler(ctx, run, input, resume)
	})
}
