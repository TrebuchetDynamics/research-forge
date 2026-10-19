package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/filetxn"
)

type Status string

const (
	Queued      Status = "queued"
	Running     Status = "running"
	Completed   Status = "completed"
	Interrupted Status = "interrupted"
	Failed      Status = "failed"
)

var (
	ErrBusy         = errors.New("a job is already running in this research directory")
	ErrNotResumable = errors.New("job is already completed")
)

type Progress struct {
	Completed int `json:"completed"`
	Total     int `json:"total"`
}

// Job is a durable, surface-independent operation. Input contains the resolved
// request, not command-line flags, file references or credentials.
type Job struct {
	ID         string          `json:"id"`
	Kind       string          `json:"kind"`
	Status     Status          `json:"status"`
	Progress   Progress        `json:"progress"`
	StopReason string          `json:"stop_reason"`
	Input      json.RawMessage `json:"input"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// Store keeps jobs under a research directory. The kernel-held lease is
// released even on SIGKILL; no PID reuse, timeout or detached daemon is involved.
// A directory has one writer because its research artifacts are shared.
type Store struct{ Dir string }

func NewStore(dir string) Store { return Store{Dir: dir} }
func (s Store) path(id string) (string, error) {
	if !strings.HasPrefix(id, "job_") || len(id) != 36 {
		return "", fmt.Errorf("invalid job ID %q", id)
	}
	if _, err := hex.DecodeString(id[4:]); err != nil {
		return "", fmt.Errorf("invalid job ID %q", id)
	}
	return filepath.Join(s.Dir, "jobs", id+".json"), nil
}
func (s Store) read(id string) (Job, error) {
	path, err := s.path(id)
	if err != nil {
		return Job{}, err
	}
	data, err := filetxn.ReadRegular(path)
	if err != nil {
		return Job{}, err
	}
	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return Job{}, fmt.Errorf("read job %s: %w", id, err)
	}
	if job.ID != id {
		return Job{}, fmt.Errorf("job ID does not match %s", path)
	}
	return job, nil
}
func (s Store) write(job Job) error {
	path, err := s.path(job.ID)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	return filetxn.Replace(path, append(data, '\n'), 0o644)
}
func (s Store) lease() (*os.File, error) {
	dir := filepath.Join(s.Dir, "jobs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, "run.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockLease(f); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

// Acquire protects research artifacts for legacy writers that do not execute a
// durable job. The caller must close the lease. Job hosts should use Run.
func (s Store) Acquire() (io.Closer, error) { return s.lease() }

func (s Store) Create(kind string, input any) (Job, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return Job{}, err
	}
	if strings.TrimSpace(kind) == "" {
		return Job{}, errors.New("job kind is required")
	}
	lease, err := s.lease()
	if err != nil {
		return Job{}, err
	}
	defer lease.Close()
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return Job{}, err
	}
	now := time.Now().UTC()
	job := Job{ID: "job_" + hex.EncodeToString(id[:]), Kind: kind, Status: Queued, Input: data, CreatedAt: now, UpdatedAt: now}
	return job, s.write(job)
}

func (s Store) list() ([]Job, error) {
	entries, err := os.ReadDir(filepath.Join(s.Dir, "jobs"))
	if os.IsNotExist(err) {
		return []Job{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []Job{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		job, err := s.read(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

// recover runs only while holding the lease, so live jobs are never interrupted
// merely because another CLI/MCP client inspected the directory.
func (s Store) recover() error {
	all, err := s.list()
	if err != nil {
		return err
	}
	for _, job := range all {
		if job.Status != Running {
			continue
		}
		job.Status, job.StopReason = Interrupted, "interrupted"
		job.UpdatedAt = time.Now().UTC()
		if err := s.write(job); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) List() ([]Job, error) {
	// Do not create directories for a read of an unused research directory.
	if _, err := os.Stat(filepath.Join(s.Dir, "jobs")); os.IsNotExist(err) {
		return []Job{}, nil
	}
	lease, err := s.lease()
	if err == nil {
		defer lease.Close()
		if err := s.recover(); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, ErrBusy) {
		return nil, err
	}
	return s.list()
}
func (s Store) Status(id string) (Job, error) {
	if _, err := s.path(id); err != nil {
		return Job{}, err
	}
	all, err := s.List()
	if err != nil {
		return Job{}, err
	}
	for _, job := range all {
		if job.ID == id {
			return job, nil
		}
	}
	return Job{}, fmt.Errorf("job %s: %w", id, os.ErrNotExist)
}

// Run is an in-process execution lease. Both CLI and server adapters supply the
// same kind-specific handler; resumption uses the saved Input and checkpoints.
type Run struct {
	store Store
	job   Job
}

func (r *Run) Job() Job { return r.job }

// SetResult saves the host-independent CLI JSON response before completion.
func (r *Run) SetResult(result json.RawMessage) error {
	if !json.Valid(result) {
		return errors.New("invalid job result JSON")
	}
	r.job.Result = append(json.RawMessage(nil), result...)
	return r.Checkpoint(r.job.Progress, r.job.StopReason)
}

// SetInput records a resolved request when a host explicitly changes a resumed
// worklist. The new inputs are durable before any additional work starts.
func (r *Run) SetInput(input any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	r.job.Input = data
	return r.Checkpoint(r.job.Progress, r.job.StopReason)
}

func (r *Run) Checkpoint(progress Progress, stopReason string) error {
	if progress.Completed < 0 || progress.Total < progress.Completed {
		return errors.New("invalid job progress")
	}
	r.job.Progress, r.job.StopReason = progress, stopReason
	r.job.UpdatedAt = time.Now().UTC()
	return r.store.write(r.job)
}
func (s Store) Run(ctx context.Context, id string, fn func(context.Context, *Run) error) (Job, error) {
	lease, err := s.lease()
	if err != nil {
		return Job{}, err
	}
	defer lease.Close()
	if err := s.recover(); err != nil {
		return Job{}, err
	}
	job, err := s.read(id)
	if err != nil {
		return Job{}, err
	}
	if job.Status == Completed {
		return job, ErrNotResumable
	}
	if fn == nil {
		return job, errors.New("job handler is required")
	}
	job.Status, job.Error, job.StopReason = Running, "", ""
	job.Result = nil
	run := &Run{store: s, job: job}
	if err := run.Checkpoint(job.Progress, ""); err != nil {
		return job, err
	}
	err = fn(ctx, run)
	run.job.Status = Completed
	if ctx.Err() != nil {
		run.job.Status, run.job.StopReason = Interrupted, "interrupted"
	} else if err != nil {
		run.job.Status, run.job.StopReason, run.job.Error = Failed, "error_threshold", err.Error()
	} else if run.job.StopReason == "interrupted" {
		run.job.Status = Interrupted
	} else if run.job.StopReason == "" {
		run.job.StopReason = "completed"
	}
	run.job.UpdatedAt = time.Now().UTC()
	if saveErr := s.write(run.job); saveErr != nil {
		err = errors.Join(err, saveErr)
	}
	return run.job, err
}
