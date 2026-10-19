package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/cli"
	"github.com/TrebuchetDynamics/research-forge/internal/jobs"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
	"github.com/TrebuchetDynamics/research-forge/internal/research"
	"github.com/TrebuchetDynamics/research-forge/internal/retrieval"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// A wrong tool mapping, lost argument or alternate result shape must break parity.
func TestToolCLIParity(t *testing.T) {
	var requests atomic.Int32
	original := http.DefaultTransport
	http.DefaultTransport = fixtureTransport(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		if r.URL.Host != "mcp-fixture.invalid" {
			return nil, fmt.Errorf("unexpected network request: %s", r.URL)
		}
		body := `{"results":[{"id":"https://openalex.org/W123","title":"Snapshot paper","publication_year":2024}]}`
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = original })
	t.Setenv("RFORGE_OPENALEX_URL", "https://mcp-fixture.invalid")
	t.Setenv("RFORGE_SOURCE_REQUEST_DELAY", "0s")
	project := filepath.Join(t.TempDir(), "project")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	session, _ := startServer(t, ctx, project)
	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, []string{"compare", "evidence_retrieve", "job_status", "research_status", "search", "source_get"}) {
		t.Fatalf("tools: %v", names)
	}
	if requests.Load() != 0 {
		t.Fatal("startup performed source work")
	}
	if _, err := os.Stat(project); !os.IsNotExist(err) {
		t.Fatalf("startup touched project: %v", err)
	}

	a, b := filepath.Join(t.TempDir(), "a"), filepath.Join(t.TempDir(), "b")
	for _, dir := range []string{a, b} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(dir, "manifest.json"), `{"queries":["snapshot"],"sources":["openalex"],"results":1,"deduped":1,"createdAt":"2026-01-01T00:00:00Z"}`)
		write(t, filepath.Join(dir, "results-deduped.jsonl"), `{"title":"Snapshot paper","year":2024,"identifiers":{"DOI":"10.1/snapshot"}}`+"\n")
	}
	index, err := retrieval.OpenSQLiteIndex(filepath.Join(project, "data", "retrieval.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := index.Rebuild([]parsing.ParsedDocument{research.ParsedTextDocument("paper-1", "Snapshot", "snapshot evidence passage", 1400)}); err != nil {
		t.Fatal(err)
	}
	index.Close()
	cases := []struct {
		name    string
		args    map[string]any
		command []string
	}{
		{"source_get", map[string]any{"id": "W123"}, []string{"paper", "W123"}},
		{"research_status", map[string]any{"dir": a}, []string{"brief", a}},
		{"compare", map[string]any{"dirs": []string{a, b}, "labels": []string{"first", "second"}}, []string{"compare", a, b, "--labels", "first,second"}},
		{"evidence_retrieve", map[string]any{"query": "snapshot"}, []string{"--project", project, "retrieve", "--query", "snapshot"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := call(t, ctx, session, tc.name, tc.args)
			want := runCLI(t, tc.command...)
			equalJSON(t, got, want)
		})
	}
	t.Run("search", func(t *testing.T) {
		session, stopServer := startServer(t, ctx, "")
		dir := filepath.Join(t.TempDir(), "search")
		initial := call(t, ctx, session, "search", map[string]any{"dir": dir, "queries": []string{"snapshot"}, "sources": []string{"openalex"}, "limit": 2, "depth": "quick"})
		var response struct {
			Data struct {
				ID string `json:"job_id"`
			}
		}
		if err := json.Unmarshal(initial, &response); err != nil {
			t.Fatal(err)
		}
		if response.Data.ID == "" {
			t.Fatalf("missing job ID: %s", initial)
		}
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		var job jobs.Job
		for {
			job, err = jobs.NewStore(dir).Status(response.Data.ID)
			// Concurrent atomic checkpoints may invalidate a read; poll for
			// a stable terminal record rather than failing on a transient read.
			if err == nil && (job.Status == jobs.Completed || job.Status == jobs.Failed) {
				break
			}
			select {
			case <-ctx.Done():
				t.Fatalf("waiting for job: %v; last read: %v; state: %+v", ctx.Err(), err, job)
			case <-ticker.C:
			}
		}
		if job.Status != jobs.Completed {
			t.Fatalf("job: %+v", job)
		}
		// Capture MCP artifacts before running the CLI against the exact same path.
		manifest := read(t, filepath.Join(dir, "manifest.json"))
		results := read(t, filepath.Join(dir, "results-deduped.jsonl"))
		t.Run("job_status", func(t *testing.T) {
			equalJSON(t, call(t, ctx, session, "job_status", map[string]any{"dir": dir, "id": job.ID}), runCLI(t, "jobs", "status", job.ID, "--dir", dir))
		})
		// Join the publisher before reusing its exact artifact directory.
		stopServer()
		cliResult := runCLI(t, "search", "batch", "--out", dir, "--query", "snapshot", "--sources", "openalex", "--limit", "2", "--depth", "quick")
		// Identity and wall-clock fields necessarily belong to separate executions.
		equalIgnoring(t, job.Result, cliResult, "job_id", "run_id")
		equalIgnoring(t, manifest, read(t, filepath.Join(dir, "manifest.json")), "createdAt")
		if !bytes.Equal(results, read(t, filepath.Join(dir, "results-deduped.jsonl"))) {
			t.Fatal("search results differ")
		}
	})
}

type fixtureTransport func(*http.Request) (*http.Response, error)

func (f fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func startServer(t *testing.T, ctx context.Context, project string) (*sdk.ClientSession, func()) {
	t.Helper()
	input, clientWriter := io.Pipe()
	clientReader, output := io.Pipe()
	var stderr bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- cli.ExecuteWithInput([]string{"--project", project, "mcp", "serve"}, input, output, &stderr)
		output.Close()
	}()
	client := sdk.NewClient(&sdk.Implementation{Name: "parity-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdk.IOTransport{Reader: clientReader, Writer: clientWriter}, nil)
	if err != nil {
		input.Close()
		output.Close()
		t.Fatalf("connect: %v", err)
	}
	stop := sync.OnceFunc(func() {
		session.Close()
		defer input.Close()
		defer output.Close()
		select {
		case code := <-done:
			if code != 0 {
				t.Errorf("serve=%d: %s", code, &stderr)
			}
		case <-time.After(5 * time.Second):
			t.Error("server did not stop")
		}
	})
	t.Cleanup(stop)
	return session, stop
}
func call(t *testing.T, ctx context.Context, s *sdk.ClientSession, name string, args map[string]any) []byte {
	t.Helper()
	result, err := s.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("%s: %+v", name, result.Content)
	}
	if result.StructuredContent == nil {
		t.Fatalf("%s missing structured JSON", name)
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func runCLI(t *testing.T, args ...string) []byte {
	t.Helper()
	var out, stderr bytes.Buffer
	if code := cli.Execute(append([]string{"--json"}, args...), &out, &stderr); code != 0 {
		t.Fatalf("CLI %v: %d %s %s", args, code, &out, &stderr)
	}
	return out.Bytes()
}
func equalJSON(t *testing.T, got, want []byte) { t.Helper(); equalIgnoring(t, got, want) }
func equalIgnoring(t *testing.T, got, want []byte, keys ...string) {
	t.Helper()
	var a, b map[string]any
	if err := json.Unmarshal(got, &a); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &b); err != nil {
		t.Fatal(err)
	}
	for _, m := range []map[string]any{a, b} {
		for _, k := range keys {
			delete(m, k)
			if d, ok := m["data"].(map[string]any); ok {
				delete(d, k)
			}
		}
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("JSON mismatch\ngot: %s\nwant: %s", got, want)
	}
}
func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
func read(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
