package oss

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// CloneRequest describes a deterministic shallow repository clone operation.
type CloneRequest struct {
	URL         string
	Destination string
	Shallow     bool
	Depth       int
}

// Args renders git clone arguments without invoking a shell.
func (r CloneRequest) Args() []string {
	args := []string{"clone"}
	if r.Shallow {
		args = append(args, "--depth", fmt.Sprintf("%d", r.Depth))
	}
	args = append(args, r.URL, r.Destination)
	return args
}

// CloneRunner runs a repository clone through an injectable boundary.
type CloneRunner interface {
	Clone(context.Context, CloneRequest) error
}

// GitCloneRunner runs git clone directly without shell interpolation.
type GitCloneRunner struct{}

// Clone executes git clone for the request.
func (GitCloneRunner) Clone(ctx context.Context, request CloneRequest) error {
	cmd := exec.CommandContext(ctx, "git", request.Args()...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// CloneResult describes a completed local clone registration.
type CloneResult struct {
	Name string
	Path string
}

// CloneRepository resolves a safe clone path and invokes the provided shallow clone runner.
func CloneRepository(ctx context.Context, projectPath, name, remoteURL string, runner CloneRunner) (CloneResult, error) {
	if runner == nil {
		return CloneResult{}, fmt.Errorf("clone runner is required")
	}
	path, err := ResolveClonePath(projectPath, name)
	if err != nil {
		return CloneResult{}, err
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return CloneResult{}, err
	}
	request := CloneRequest{URL: remoteURL, Destination: path, Shallow: true, Depth: 1}
	if err := runner.Clone(ctx, request); err != nil {
		return CloneResult{}, err
	}
	return CloneResult{Name: name, Path: path}, nil
}
