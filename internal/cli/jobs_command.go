package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/TrebuchetDynamics/research-forge/internal/jobs"
	"github.com/TrebuchetDynamics/research-forge/internal/search"
)

const usageJobs = "usage: rforge jobs list|status <id>|resume <id> [--dir <research-dir>]"

func executeJobs(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", usageJobs)
	}
	command, dir, id := args[0], ".", ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--dir" && i+1 < len(args) {
			i++
			dir = args[i]
		} else if id == "" && args[i] != "--dir" {
			id = args[i]
		} else {
			return writeError(stdout, stderr, opts, 2, "usage", usageJobs)
		}
	}
	store := jobs.NewStore(dir)
	if command == "list" && id == "" {
		all, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "jobs_list_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"jobs": all})
		}
		for _, job := range all {
			fmt.Fprintf(stdout, "%s\t%s\t%s\t%d/%d\t%s\n", job.ID, job.Kind, job.Status, job.Progress.Completed, job.Progress.Total, job.StopReason)
		}
		return 0
	}
	if (command != "status" && command != "resume") || id == "" {
		return writeError(stdout, stderr, opts, 2, "usage", usageJobs)
	}
	job, err := store.Status(id)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "job_read_failed", err.Error())
	}
	if command == "status" {
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"job": job})
		}
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%d/%d\t%s\n", job.ID, job.Kind, job.Status, job.Progress.Completed, job.Progress.Total, job.StopReason)
		return 0
	}
	if job.Kind != search.BatchJobKind {
		return writeError(stdout, stderr, opts, 2, "unknown_job_kind", job.Kind)
	}
	if job.Status == jobs.Completed {
		return writeError(stdout, stderr, opts, 2, "job_not_resumable", jobs.ErrNotResumable.Error())
	}
	var input search.BatchJobInput
	if err := json.Unmarshal(job.Input, &input); err != nil {
		return writeError(stdout, stderr, opts, 1, "job_input_failed", err.Error())
	}
	opts.Project = input.Project
	batch := batchOptionsFromJob(input, dir, true)
	return executeSearchBatchPrepared(batch, input.Queries, stdout, stderr, opts, id)
}

func batchOptionsFromJob(input search.BatchJobInput, dir string, resume bool) searchBatchOptions {
	return searchBatchOptions{Queries: input.Queries, Sources: input.Sources, Limit: input.Limit, OutDir: dir, Resume: resume, ContinueOnError: input.ContinueOnError, WriteStats: input.WriteStats, FetchPDFs: input.FetchPDFs, Depth: input.Depth, MaxRequests: input.MaxRequests}
}

func runSearchBatchJob(batch searchBatchOptions, queries []string, configurationErrors map[string]string, stdout, stderr io.Writer, opts globalOptions, id string) int {
	project := opts.Project
	if project != "" {
		var err error
		project, err = filepath.Abs(project)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "job_input_failed", err.Error())
		}
	}
	input := search.BatchJobInput{Queries: queries, Sources: batch.Sources, Limit: batch.Limit, Depth: batch.Depth, MaxRequests: batch.MaxRequests, ContinueOnError: batch.ContinueOnError, WriteStats: batch.WriteStats, FetchPDFs: batch.FetchPDFs, Project: project}
	code := 0
	_, err := search.RunBatchJob(context.Background(), batch.OutDir, id, input, batch.Resume, func(_ context.Context, run *jobs.Run, saved search.BatchJobInput, resume bool) error {
		options := batchOptionsFromJob(saved, batch.OutDir, resume)
		options.QueriesFile = batch.QueriesFile
		opts.Project = saved.Project
		code = executeSearchBatchRun(options, saved.Queries, configurationErrors, stdout, stderr, opts, run)
		if code != 0 && run.Job().StopReason != "interrupted" {
			return fmt.Errorf("search batch exited with code %d", code)
		}
		return nil
	})
	if err != nil && code == 0 {
		return writeError(stdout, stderr, opts, 1, "job_run_failed", err.Error())
	}
	return code
}
