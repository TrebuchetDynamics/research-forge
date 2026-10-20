package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
)

// Serve real HTTP locally, using the worker's reserved port range when present.
func workflowFixtureServer(t *testing.T) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/works", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"meta":{"count":2},"results":[
			{"id":"https://openalex.org/W1","doi":"https://doi.org/10.1000/example","title":"Scaling Study","publication_year":2023,"cited_by_count":100,"primary_location":{"landing_page_url":"https://github.com/example/scaling"},"open_access":{"is_oa":true},"abstract_inverted_index":{"Scaling":[0],"experiment":[1]}},
			{"id":"https://openalex.org/W2","title":"Earlier Study","publication_year":2020,"cited_by_count":1}]}`)
	})
	mux.HandleFunc("/works/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ok","message":{"DOI":"10.1000/example","title":["Scaling Study"]}}`)
	})
	mux.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"doi":"10.1000/example","is_oa":true,"oa_status":"gold","best_oa_location":{"url_for_pdf":"https://example.org/paper.pdf","license":"cc-by"}}`)
	})
	address := "127.0.0.1:0"
	var listener net.Listener
	var err error
	if base, _ := strconv.Atoi(os.Getenv("AUTOWORK_PORT_BASE")); base > 0 {
		for port := base; port < base+100; port++ {
			listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
			if err == nil {
				break
			}
		}
	} else {
		listener, err = net.Listen("tcp", address)
	}
	if err != nil {
		t.Fatal(err)
	}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: mux}}
	server.Start()
	t.Cleanup(server.Close)
	for _, name := range []string{"OPENALEX", "CROSSREF", "UNPAYWALL", "ARXIV", "SEMANTIC_SCHOLAR"} {
		t.Setenv("RFORGE_"+name+"_URL", server.URL)
	}
	t.Setenv("RFORGE_UNPAYWALL_EMAIL", "researcher@example.org")
	t.Setenv("RFORGE_SOURCE_REQUEST_DELAY", "0s")
}

func TestRankPersistsRankingsWithoutReplacingProvenance(t *testing.T) {
	workflowFixtureServer(t)
	for _, jsonOutput := range []bool{false, true} {
		t.Run(fmt.Sprintf("json=%t", jsonOutput), func(t *testing.T) {
			dir := t.TempDir()
			original := []byte("{\"original\": true}\n")
			provPath := filepath.Join(dir, "provenance.json")
			if err := os.WriteFile(provPath, original, 0o644); err != nil {
				t.Fatal(err)
			}
			outPath := filepath.Join(dir, "rankings.json")
			args := []string{"rank", "scaling", "--sources", "openalex", "--out", outPath}
			if jsonOutput {
				args = append([]string{"--json"}, args...)
			}
			var out, stderr bytes.Buffer
			if code := Execute(args, &out, &stderr); code != 0 {
				t.Fatalf("rank exit %d: %s %s", code, &out, &stderr)
			}
			got, err := os.ReadFile(provPath)
			if err != nil || !bytes.Equal(got, original) {
				t.Errorf("existing provenance changed: %s, %v", got, err)
			}
			data, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatal(err)
			}
			var result struct {
				Rankings []struct {
					Rank  int     `json:"rank"`
					Title string  `json:"title"`
					Score float64 `json:"score"`
				} `json:"rankings"`
			}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatal(err)
			}
			if len(result.Rankings) != 2 || result.Rankings[0].Title != "Scaling Study" || result.Rankings[1].Title != "Earlier Study" || result.Rankings[1].Rank != 2 || result.Rankings[0].Score <= result.Rankings[1].Score {
				t.Errorf("incomplete or unordered rankings: %s", data)
			}
			assertEngineReceipt(t, dir, "rankings.json")
			out.Reset()
			if code := Execute([]string{"provenance", "validate", outPath + ".provenance.json"}, &out, &stderr); code != 0 {
				t.Errorf("rank provenance invalid: %s %s", &out, &stderr)
			}
			if code := Execute([]string{"rank", "scaling", "--sources", "openalex", "--out", provPath}, &out, &stderr); code == 0 {
				t.Error("rank must reject provenance.json as an output destination")
			}
			got, err = os.ReadFile(provPath)
			if err != nil || !bytes.Equal(got, original) {
				t.Errorf("explicit --out replaced existing provenance: %s, %v", got, err)
			}
		})
	}
}

func TestRankDefaultOutputHasEngineReceipt(t *testing.T) {
	workflowFixtureServer(t)
	t.Chdir(t.TempDir())
	var out, stderr bytes.Buffer
	if code := Execute([]string{"--json", "rank", "scaling", "--sources", "openalex"}, &out, &stderr); code != 0 {
		t.Fatalf("rank=%d: %s %s", code, &out, &stderr)
	}
	assertEngineReceipt(t, "rankings", "rankings.json")
}

func TestRankSurfacesWriteErrors(t *testing.T) {
	workflowFixtureServer(t)
	for _, target := range []string{"output", "provenance"} {
		t.Run(target, func(t *testing.T) {
			outPath := filepath.Join(t.TempDir(), "rankings.json")
			blocked := outPath
			if target == "provenance" {
				blocked += ".provenance.json"
			}
			if err := os.Mkdir(blocked, 0o755); err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			if code := Execute([]string{"--json", "rank", "scaling", "--sources", "openalex", "--out", outPath}, &out, &stderr); code == 0 || !strings.Contains(out.String(), "rank_write_failed") {
				t.Fatalf("expected rank write failure, got %d: %s %s", code, &out, &stderr)
			}
			receipts, err := filepath.Glob(filepath.Join(filepath.Dir(outPath), "provenance", "runs", "*.json"))
			if err != nil || len(receipts) != 0 {
				t.Fatalf("failed rank left successful receipts: %v (%v)", receipts, err)
			}
		})
	}
}

func TestWorkflowCommandsRecordProjectHistory(t *testing.T) {
	workflowFixtureServer(t)
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	for _, dir := range []string{a, b} {
		writeFixtureSearchDir(t, dir, "scaling", []map[string]any{{"title": "Scaling Study", "identifiers": map[string]any{"DOI": "10.1000/example"}}})
	}
	for _, args := range [][]string{
		{"rank", "scaling", "--sources", "openalex"},
		{"paper", "10.1000/example"},
		{"draft", "10.1000/example"},
		{"draft", "--search-dir", a},
		{"replicate", "10.1000/example"},
		{"lit", "--research-dir", root},
		{"history", "search", "scaling", "--research-dir", root},
		{"brief", a},
		{"compare", a, b},
		{"note", "add", "--paper", "10.1000/example", "--note", "reviewed"},
	} {
		t.Run(strings.Join(args[:2], " "), func(t *testing.T) {
			project := t.TempDir()
			var out, stderr bytes.Buffer
			if code := Execute(append([]string{"--project", project}, args...), &out, &stderr); code != 0 {
				t.Fatalf("command exit %d: %s %s", code, &out, &stderr)
			}
			events, err := provenance.Read(project)
			if err != nil || len(events) != 1 {
				t.Fatalf("expected one command event, got %+v, %v", events, err)
			}
			if events[0].Action != "cli.command" || !strings.HasPrefix(events[0].Inputs["command"].(string), args[0]) || events[0].Outputs["exitCode"] != float64(0) {
				t.Errorf("unexpected command event: %+v", events[0])
			}
		})
	}
}

func TestWorkflowCommandSurfacesHistoryWriteFailure(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "provenance"), []byte("blocked"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	writeFixtureSearchDir(t, dir, "scaling", nil)
	var out, stderr bytes.Buffer
	code := Execute([]string{"--json", "--project", project, "draft", "--search-dir", dir}, &out, &stderr)
	if code == 0 || !strings.Contains(out.String(), "provenance_append_failed") || strings.Contains(out.String(), `"ok":true`) {
		t.Fatalf("expected history failure without success output, got %d: %s %s", code, &out, &stderr)
	}
}

func TestPaperAndDraftResolveOffline(t *testing.T) {
	workflowFixtureServer(t)
	for _, command := range []string{"paper", "draft"} {
		t.Run(command, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), command+".out")
			var out, stderr bytes.Buffer
			if code := Execute([]string{"--json", command, "10.1000/example", "--out", path}, &out, &stderr); code != 0 {
				t.Fatalf("exit %d: %s %s", code, &out, &stderr)
			}
			data, err := os.ReadFile(path)
			if err != nil || !bytes.Contains(data, []byte("Scaling Study")) || !bytes.Contains(data, []byte("10.1000/example")) {
				t.Fatalf("missing resolved paper in output: %s, %v", data, err)
			}
			if !strings.Contains(out.String(), "Scaling Study") {
				t.Errorf("missing resolved title in JSON: %s", &out)
			}
		})
	}
}

// A flag where the topic belongs must be a usage error. It used to rank the
// literal string "--limit", exit 0, and present the result as a real ranking.
func TestRankRejectsFlagAsTopic(t *testing.T) {
	workflowFixtureServer(t)

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"rank", "--limit", "5"}, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (usage); stdout = %s", code, stdout.String())
	}
	if !strings.Contains(stdout.String()+stderr.String(), "usage: rforge rank <topic>") {
		t.Errorf("error does not show usage: %s", stdout.String()+stderr.String())
	}
	if strings.Contains(stdout.String(), "PaperRank: --limit") {
		t.Errorf("ranked the literal flag as a topic: %s", stdout.String())
	}
}

// An unknown flag must be rejected, not have its value treated as the paper.
// `draft --depth quick` used to draft the paper matching "quick" and exit 0.
func TestDraftRejectsUnknownFlagValueAsPaper(t *testing.T) {
	workflowFixtureServer(t)

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"draft", "--depth", "quick"}, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (usage); stdout = %s", code, stdout.String())
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "--depth") {
		t.Errorf("error does not name the unknown flag: %s", combined)
	}
	if strings.Contains(stdout.String(), "# Research Note") {
		t.Errorf("drafted a paper from a flag value: %s", stdout.String())
	}
}

// An empty identifier must not resolve to an arbitrary paper.
func TestPaperRejectsEmptyIdentifier(t *testing.T) {
	workflowFixtureServer(t)

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"paper", ""}, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (usage); stdout = %s", code, stdout.String())
	}
	if strings.Contains(stdout.String(), "✓") {
		t.Errorf("resolved a paper from an empty identifier: %s", stdout.String())
	}
}

func TestReplicateRejectsEmptyIdentifier(t *testing.T) {
	workflowFixtureServer(t)

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	code := Execute([]string{"replicate", ""}, stdout, stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (usage); stdout = %s", code, stdout.String())
	}
	if strings.Contains(stdout.String(), "# Replication Plan") {
		t.Errorf("replicated a paper from an empty identifier: %s", stdout.String())
	}
}
