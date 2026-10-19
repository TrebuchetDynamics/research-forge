package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/jobs"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	adapter "github.com/TrebuchetDynamics/research-forge/internal/mcp"
	"github.com/TrebuchetDynamics/research-forge/internal/research"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Catch synchronous searches and request-scoped jobs: tool return must not wait
// for the provider, but session disconnect must cancel and checkpoint the job.
func TestSearchLifetimeBelongsToServer(t *testing.T) {
	started, canceled := make(chan struct{}), make(chan struct{})
	original := http.DefaultTransport
	http.DefaultTransport = fixtureTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "blocked-fixture.invalid" {
			return nil, fmt.Errorf("unexpected request: %s", r.URL)
		}
		close(started)
		<-r.Context().Done()
		close(canceled)
		return nil, r.Context().Err()
	})
	t.Cleanup(func() { http.DefaultTransport = original })
	t.Setenv("RFORGE_OPENALEX_URL", "https://blocked-fixture.invalid")
	t.Setenv("RFORGE_SOURCE_REQUEST_DELAY", "0s")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	session, stopServer := startServer(t, ctx, "")
	dir := filepath.Join(t.TempDir(), "research")
	requestCtx, cancelRequest := context.WithCancel(ctx)
	call(t, requestCtx, session, "search", map[string]any{"dir": dir, "queries": []string{"blocked"}, "sources": []string{"openalex"}})
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	cancelRequest()
	// A second call confirms that the session remains usable after the first
	// call's context ended, while the provider request is still blocked.
	all, err := jobs.NewStore(dir).List()
	if err != nil || len(all) != 1 {
		t.Fatalf("jobs: %+v %v", all, err)
	}
	status, err := session.CallTool(ctx, &sdk.CallToolParams{Name: "job_status", Arguments: map[string]any{"dir": dir, "id": all[0].ID}})
	if err != nil || status.IsError {
		t.Fatalf("status: %+v %v", status, err)
	}
	select {
	case <-canceled:
		t.Fatal("tool request cancellation stopped the job")
	default:
	}
	stopServer()
	select {
	case <-canceled:
	case <-ctx.Done():
		t.Fatal("disconnect did not cancel provider")
	}
	job, err := jobs.NewStore(dir).Status(all[0].ID)
	if err != nil || job.Status != jobs.Interrupted || job.StopReason != "interrupted" {
		t.Fatalf("job was not checkpointed: %+v %v", job, err)
	}
}

// The SDK's stdio transport does not propagate root cancellation to tool
// requests. The adapter must link the two lifetimes for provider lookups.
func TestSourceGetStopsOnServerCancellation(t *testing.T) {
	started, canceled := make(chan struct{}), make(chan struct{})
	original := http.DefaultTransport
	http.DefaultTransport = fixtureTransport(func(r *http.Request) (*http.Response, error) {
		close(started)
		<-r.Context().Done()
		close(canceled)
		return nil, r.Context().Err()
	})
	t.Cleanup(func() { http.DefaultTransport = original })
	t.Setenv("RFORGE_OPENALEX_URL", "https://cancel-fixture.invalid")
	t.Setenv("RFORGE_SOURCE_REQUEST_DELAY", "0s")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	root, stop := context.WithCancel(ctx)
	defer stop()
	serverTransport, clientTransport := sdk.NewInMemoryTransports()
	done := make(chan error, 1)
	go func() {
		done <- adapter.Serve(root, serverTransport, adapter.Config{Build: research.BuildInfo{Version: "test"}})
	}()
	client := sdk.NewClient(&sdk.Implementation{Name: "cancel-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		session.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("server did not stop")
		}
	})
	go func() {
		_, _ = session.CallTool(ctx, &sdk.CallToolParams{Name: "source_get", Arguments: map[string]any{"id": "W123"}})
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	stop()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("server cancellation did not cancel source_get")
	}
}

func TestInvalidToolInputsDoNotStartWork(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	project := filepath.Join(t.TempDir(), "project")
	session, _ := startServer(t, ctx, project)
	dir := filepath.Join(t.TempDir(), "invalid")
	cases := []struct {
		name string
		args map[string]any
	}{
		{"search", map[string]any{"dir": dir, "queries": []string{"q"}, "sources": []string{"not-a-source"}}},
		{"search", map[string]any{"dir": dir, "queries": []string{"q"}, "sources": []string{"openalex"}, "depth": "unbounded"}},
		{"search", map[string]any{"dir": dir, "queries": []string{" "}, "sources": []string{"openalex"}}},
		{"source_get", map[string]any{"id": " "}},
		{"job_status", map[string]any{"dir": dir, "id": "../../outside"}},
		{"research_status", map[string]any{"dir": " "}},
		{"compare", map[string]any{"dirs": []string{dir}}},
		{"evidence_retrieve", map[string]any{"query": "q", "backend": "arbitrary"}},
	}
	for _, tc := range cases {
		result, err := session.CallTool(ctx, &sdk.CallToolParams{Name: tc.name, Arguments: tc.args})
		if err != nil {
			t.Fatal(err)
		}
		if !result.IsError {
			t.Fatalf("%s accepted invalid input: %+v", tc.name, result)
		}
	}
	all, err := jobs.NewStore(dir).List()
	if err != nil || len(all) != 0 {
		t.Fatalf("invalid calls created work: %+v %v", all, err)
	}
}

// Project library publication (including rollback) has one writer per server,
// even when calls target different research directories.
func TestSearchSerializesSharedProject(t *testing.T) {
	firstStarted, secondStarted, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	releaseFirst := sync.OnceFunc(func() { close(release) })
	defer releaseFirst()
	original := http.DefaultTransport
	http.DefaultTransport = fixtureTransport(func(r *http.Request) (*http.Response, error) {
		query := r.URL.Query().Get("search")
		if r.URL.Host != "project-fixture.invalid" {
			return nil, fmt.Errorf("unexpected request: %s", r.URL)
		}
		if query == "first" {
			close(firstStarted)
			select {
			case <-release:
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
		} else {
			close(secondStarted)
		}
		body := fmt.Sprintf(`{"results":[{"id":"https://openalex.org/W%s","title":"%s paper","doi":"https://doi.org/10.1000/%s"}]}`, query, query, query)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = original })
	t.Setenv("RFORGE_OPENALEX_URL", "https://project-fixture.invalid")
	t.Setenv("RFORGE_SOURCE_REQUEST_DELAY", "0s")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	project := t.TempDir()
	session, _ := startServer(t, ctx, project)
	dirs := []string{filepath.Join(t.TempDir(), "first"), filepath.Join(t.TempDir(), "second")}
	var ids []string
	for i, query := range []string{"first", "second"} {
		data := call(t, ctx, session, "search", map[string]any{"dir": dirs[i], "queries": []string{query}, "sources": []string{"openalex"}})
		var response struct {
			Data struct {
				ID string `json:"job_id"`
			}
		}
		if err := json.Unmarshal(data, &response); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, response.Data.ID)
		if i == 0 {
			select {
			case <-firstStarted:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		}
	}
	select {
	case <-secondStarted:
		t.Fatal("searches sharing a project ran concurrently")
	case <-time.After(50 * time.Millisecond):
	}
	releaseFirst()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for i, dir := range dirs {
		for {
			job, err := jobs.NewStore(dir).Status(ids[i])
			if err == nil && job.Status == jobs.Completed {
				break
			}
			if job.Status == jobs.Failed {
				t.Fatalf("search failed: %+v", job)
			}
			select {
			case <-ticker.C:
			case <-ctx.Done():
				t.Fatalf("search did not finish: %+v; last read: %v", job, err)
			}
		}
	}
	store, err := library.OpenStore(filepath.Join(project, "data", "library.json"))
	if err != nil {
		t.Fatal(err)
	}
	records, err := store.List()
	if err != nil || len(records) != 2 {
		t.Fatalf("shared project lost an import: %+v %v", records, err)
	}
}
