package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/TrebuchetDynamics/research-forge/internal/jobs"
	"github.com/TrebuchetDynamics/research-forge/internal/mcp"
	"github.com/TrebuchetDynamics/research-forge/internal/research"
	"github.com/TrebuchetDynamics/research-forge/internal/search"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func executeMCP(args []string, stdin io.Reader, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 1 || args[0] != "serve" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge [--project <path>] mcp serve")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	reader, ok := stdin.(io.ReadCloser)
	if !ok {
		reader = io.NopCloser(stdin)
	}
	cfg := mcp.Config{
		Project:       opts.Project,
		Build:         research.BuildInfo{Version: Version, Commit: Commit, Date: Date},
		OpenRetrieval: openRetrievalBackend,
		PrepareSearch: func(dir string, input search.BatchJobInput) (search.BatchJobInput, error) {
			input.Queries = uniqueNonEmptyStrings(input.Queries)
			input.Sources = splitSearchBatchList(strings.Join(input.Sources, ","))
			if input.Project != "" {
				var err error
				input.Project, err = filepath.Abs(input.Project)
				if err != nil {
					return input, err
				}
			}
			_, err := validateSearchBatch(batchOptionsFromJob(input, dir, false), input.Queries, globalOptions{Project: input.Project})
			return input, err
		},
		RunSearch: func(ctx context.Context, run *jobs.Run, input search.BatchJobInput, dir string, resume bool) error {
			batch := batchOptionsFromJob(input, dir, resume)
			options := globalOptions{JSON: true, Project: input.Project}
			configurationErrors, err := validateSearchBatch(batch, input.Queries, options)
			if err != nil {
				return err
			}
			var out, diagnostics bytes.Buffer
			code := executeSearchBatchRunContext(ctx, func() syscall.Signal { return 0 }, batch, input.Queries, configurationErrors, &out, &diagnostics, options, run)
			if code != 0 && run.Job().StopReason != "interrupted" {
				return fmt.Errorf("search batch exited with code %d: %s%s", code, &out, &diagnostics)
			}
			return nil
		},
	}
	if err := mcp.Serve(ctx, &sdk.IOTransport{Reader: reader, Writer: nonClosingWriter{stdout}}, cfg); err != nil && !errors.Is(err, context.Canceled) {
		// stdout belongs exclusively to JSON-RPC, even when --json was supplied.
		fmt.Fprintf(stderr, "mcp serve: %v\n", err)
		return 1
	}
	return 0
}

type nonClosingWriter struct{ io.Writer }

func (nonClosingWriter) Close() error { return nil }
