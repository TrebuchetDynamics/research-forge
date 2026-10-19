// Package mcp is the stdio MCP adapter for the shared research engine.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/claims"
	"github.com/TrebuchetDynamics/research-forge/internal/jobs"
	"github.com/TrebuchetDynamics/research-forge/internal/research"
	"github.com/TrebuchetDynamics/research-forge/internal/retrieval"
	"github.com/TrebuchetDynamics/research-forge/internal/search"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Config supplies host policy and the same artifact publisher/backend factory
// used by the CLI. Constructing a server never opens a store or a connector.
type Config struct {
	Project       string
	Build         research.BuildInfo
	PrepareSearch func(string, search.BatchJobInput) (search.BatchJobInput, error)
	RunSearch     func(context.Context, *jobs.Run, search.BatchJobInput, string, bool) error
	OpenRetrieval func(string, string) (retrieval.SearchAdapter, error)
}

type searchInput struct {
	Dir             string   `json:"dir" jsonschema:"research output directory"`
	Queries         []string `json:"queries" jsonschema:"search queries"`
	Sources         []string `json:"sources" jsonschema:"source names or CLI presets"`
	Limit           int      `json:"limit,omitempty" jsonschema:"results per query, default 25"`
	Depth           string   `json:"depth,omitempty" jsonschema:"quick, standard (default), or comprehensive"`
	MaxRequests     int      `json:"max_requests,omitempty"`
	ContinueOnError bool     `json:"continue_on_error,omitempty"`
	Stats           bool     `json:"stats,omitempty"`
}
type jobInput struct {
	Dir string `json:"dir"`
	ID  string `json:"id"`
}
type sourceInput struct {
	ID string `json:"id" jsonschema:"DOI, arXiv ID, OpenAlex ID, URL, or title"`
}
type statusInput struct {
	Dir string `json:"dir"`
	Out string `json:"out,omitempty"`
}
type compareInput struct {
	Dirs   []string `json:"dirs"`
	Labels []string `json:"labels,omitempty"`
	Out    string   `json:"out,omitempty"`
}
type claimCheckInput struct {
	ID              string             `json:"id" jsonschema:"stored claim ID"`
	Project         string             `json:"project,omitempty"`
	SemanticSupport *claims.Assessment `json:"semantic_support,omitempty" jsonschema:"agent-reported assessment with agent and model identity; never an engine or human decision"`
}

type retrieveInput struct {
	Query   string `json:"query"`
	Backend string `json:"backend,omitempty"`
	Project string `json:"project,omitempty"`
}

// Serve owns a single session and its in-process jobs. Disconnect cancels and
// joins running searches so their checkpoints are resumable through the CLI.
func Serve(ctx context.Context, transport sdk.Transport, cfg Config) error {
	ctx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	projectWriter := make(chan struct{}, 1)
	defer func() { cancel(); workers.Wait() }()
	server := sdk.NewServer(&sdk.Implementation{Name: "rforge", Version: cfg.Build.Version}, nil)
	service := research.Service{Build: cfg.Build}
	addTool(server, "search", "Start a durable search batch; returns immediately with a job ID. Poll job_status for the CLI JSON result and artifact references.", func(_ context.Context, in searchInput) (any, error) {
		if strings.TrimSpace(in.Dir) == "" || len(in.Queries) == 0 || len(in.Sources) == 0 {
			return nil, errors.New("dir, queries and sources are required")
		}
		if in.Limit == 0 {
			in.Limit = 25
		}
		if in.Limit < 1 || in.MaxRequests < 0 {
			return nil, errors.New("limit must be positive and max_requests nonnegative")
		}
		if _, err := search.DepthBudget(in.Depth); err != nil {
			return nil, err
		}
		if cfg.PrepareSearch == nil || cfg.RunSearch == nil {
			return nil, errors.New("search host is not configured")
		}
		input, err := cfg.PrepareSearch(in.Dir, search.BatchJobInput{Queries: in.Queries, Sources: in.Sources, Limit: in.Limit, Depth: in.Depth, MaxRequests: in.MaxRequests, ContinueOnError: in.ContinueOnError, WriteStats: in.Stats, Project: cfg.Project})
		if err != nil {
			return nil, err
		}
		job, err := jobs.NewStore(in.Dir).Create(search.BatchJobKind, input)
		if err != nil {
			return nil, err
		}
		workers.Add(1)
		go func() {
			defer workers.Done()
			if cfg.Project != "" {
				// Library snapshot/import/rollback is a read-modify-write
				// transaction shared even by distinct output directories.
				// Serialize the whole publisher, not merely the import call.
				select {
				case projectWriter <- struct{}{}:
					defer func() { <-projectWriter }()
				case <-ctx.Done():
					return
				}
			}
			// A status reader (or another writer) may win the lease between
			// registration and execution. Keep the accepted job queued until
			// the lease is available, rather than silently abandoning it.
			for {
				_, err := search.RunBatchJob(ctx, in.Dir, job.ID, input, false, func(ctx context.Context, run *jobs.Run, saved search.BatchJobInput, resume bool) error {
					return cfg.RunSearch(ctx, run, saved, in.Dir, resume)
				})
				if !errors.Is(err, jobs.ErrBusy) {
					return
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(10 * time.Millisecond):
				}
			}
		}()
		return map[string]any{"job_id": job.ID, "out": in.Dir}, nil
	})
	addTool(server, "job_status", "Inspect a durable job, progress, stop reason and completed CLI JSON result.", func(_ context.Context, in jobInput) (any, error) {
		if strings.TrimSpace(in.Dir) == "" {
			return nil, errors.New("dir is required")
		}
		job, err := jobs.NewStore(in.Dir).Status(in.ID)
		return map[string]any{"job": job}, err
	})
	addTool(server, "source_get", "Resolve paper metadata with the shared paper resolver; no full documents.", func(toolCtx context.Context, in sourceInput) (any, error) {
		if strings.TrimSpace(in.ID) == "" {
			return nil, errors.New("id is required")
		}
		toolCtx, cancel := context.WithTimeout(toolCtx, 90*time.Second)
		defer cancel()
		stop := context.AfterFunc(ctx, cancel)
		defer stop()
		result, err := sources.ResolvePaper(toolCtx, sources.NewHTTPClient(sources.HTTPClientOptions{}), in.ID)
		return map[string]any{"resolution": result}, err
	})
	addTool(server, "research_status", "Return a compact research brief, equivalent to brief --json.", func(_ context.Context, in statusInput) (any, error) {
		if strings.TrimSpace(in.Dir) == "" {
			return nil, errors.New("dir is required")
		}
		return service.Brief(research.BriefOptions{SearchDir: in.Dir, Out: in.Out})
	})
	addTool(server, "compare", "Compare search directories, equivalent to compare --json.", func(_ context.Context, in compareInput) (any, error) {
		return service.Compare(research.CompareOptions{Dirs: in.Dirs, Labels: in.Labels, Out: in.Out})
	})
	addTool(server, "evidence_retrieve", "Retrieve bounded evidence passages from an existing index, equivalent to retrieve --json.", func(_ context.Context, in retrieveInput) (any, error) {
		if strings.TrimSpace(in.Query) == "" {
			return nil, errors.New("query is required")
		}
		if in.Project == "" {
			in.Project = cfg.Project
		}
		if in.Project == "" {
			return nil, errors.New("project is required")
		}
		if in.Backend == "" {
			in.Backend = "sqlite"
		}
		switch in.Backend {
		case "sqlite", "opensearch", "qdrant", "hybrid":
		default:
			return nil, fmt.Errorf("unknown retrieval backend %q", in.Backend)
		}
		if cfg.OpenRetrieval == nil {
			return nil, errors.New("retrieval host is not configured")
		}
		index, err := cfg.OpenRetrieval(in.Project, in.Backend)
		if err != nil {
			return nil, err
		}
		defer index.Close()
		results, err := index.Retrieve(in.Query)
		return map[string]any{"results": results, "backend": in.Backend}, err
	})
	addTool(server, "claim_check", "Check source identity, content integrity and passage match against stored offline evidence, equivalent to claim check --json. Optional semantic_support is agent_reported. Preserves but cannot set human approval; no facet asserts truth.", func(_ context.Context, in claimCheckInput) (any, error) {
		if in.Project == "" {
			in.Project = cfg.Project
		}
		var semantic *claims.SemanticSupport
		if in.SemanticSupport != nil {
			value, err := claims.NewSemanticSupport(*in.SemanticSupport)
			if err != nil {
				return nil, err
			}
			semantic = &value
		}
		return claims.CheckProject(claims.ProjectCheckOptions{Project: in.Project, ID: in.ID, Timestamp: time.Now().UTC(), SemanticSupport: semantic})
	})
	return server.Run(ctx, transport)
}

// Preserve the CLI --json envelope in both structuredContent and text content.
func addTool[In any](server *sdk.Server, name, description string, run func(context.Context, In) (any, error)) {
	sdk.AddTool(server, &sdk.Tool{Name: name, Description: description}, func(ctx context.Context, _ *sdk.CallToolRequest, in In) (*sdk.CallToolResult, map[string]any, error) {
		data, err := run(ctx, in)
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"ok": true, "data": data}, nil
	})
}
