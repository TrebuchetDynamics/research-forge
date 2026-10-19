package research_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/research"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

func session(t *testing.T, root, name, papers string) string {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for file, data := range map[string]string{
		"manifest.json":         `{"queries":["scaling"],"sources":["openalex"],"results":2,"deduped":2,"createdAt":"2026-01-01T00:00:00Z"}`,
		"results-deduped.jsonl": papers,
	} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// Catches an extraction that drops ranking, graph boosts, deduplication or the receipt.
func TestServiceRank(t *testing.T) {
	dir := t.TempDir()
	graph := filepath.Join(dir, "graph.json")
	if err := os.WriteFile(graph, []byte(`{"edges":[{"source":"a","target":"10.1/new"},{"source":"b","target":"10.1/new"}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	s := research.Service{Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }, Connector: func(name string) (research.SearchConnector, bool) { return rankedSource{}, name == "openalex" }}
	out := filepath.Join(dir, "rankings.json")
	for _, tc := range []struct{ limit, total int }{{3, 2}, {1, 1}} {
		got, err := s.Rank(context.Background(), research.RankOptions{Topic: "scaling", Sources: []string{"openalex", "openalex"}, Limit: tc.limit, GraphPath: graph, Out: out})
		if err != nil {
			t.Fatal(err)
		}
		if got.Total != tc.total || got.Rankings[0].Title != "New" || got.Rankings[0].Score != 4.3 {
			t.Fatalf("limit %d: rank = %+v", tc.limit, got)
		}
	}
	receipt, err := provenance.Validate(out + ".provenance.json")
	if err != nil || receipt.Level != provenance.EngineVerified {
		t.Fatalf("receipt = %+v, %v", receipt, err)
	}
}

type rankedSource struct{}

func (rankedSource) Search(context.Context, sources.SourceQuery) (sources.SourceResponse, error) {
	return sources.SourceResponse{Records: []sources.SourceRecord{
		{SourceID: "old", Title: "Old", Year: 2020, Metadata: map[string]string{"cited_by_count": "1"}},
		{SourceID: "new", Title: "New", Year: 2026, OpenAccess: true, Identifiers: sources.Identifiers{DOI: "10.1/new"}},
	}}, nil
}

// Catches loss of brief metadata, year ordering or artifact persistence.
func TestServiceBrief(t *testing.T) {
	dir := session(t, t.TempDir(), "a", `{"title":"Old","year":2020}
{"title":"New","year":2026,"identifiers":{"DOI":"10.1/new"}}`)
	out := filepath.Join(t.TempDir(), "brief.md")
	got, err := (research.Service{}).Brief(research.BriefOptions{SearchDir: dir, Out: out})
	if err != nil {
		t.Fatal(err)
	}
	if got.Topic != "scaling" || got.TotalRecords != 2 || got.Papers[0].Title != "New" {
		t.Fatalf("brief = %+v", got)
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data) != got.Markdown || !strings.Contains(got.Markdown, "https://doi.org/10.1/new") {
		t.Fatalf("brief artifact = %s, %v", data, err)
	}
}

// Catches incorrect identifier union/membership and loss of comparison output.
func TestServiceCompare(t *testing.T) {
	root := t.TempDir()
	a := session(t, root, "a", `{"title":"Shared","identifiers":{"DOI":"10.1/shared"}}
{"title":"Only A","identifiers":{"ArXivID":"2301.00001"}}`)
	b := session(t, root, "b", `{"title":"Shared","identifiers":{"DOI":"10.1/shared"}}`)
	out := filepath.Join(root, "comparison.md")
	got, err := (research.Service{}).Compare(research.CompareOptions{Dirs: []string{a, b}, Labels: []string{"first", "second"}, Out: out})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 2 || got.Papers[0].ID != "10.1/shared" || strings.Join(got.Papers[0].In, ",") != "first,second" || got.Papers[1].In[0] != "first" {
		t.Fatalf("compare = %+v", got)
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data) != got.Markdown {
		t.Fatalf("comparison artifact = %s, %v", data, err)
	}
}

// Catches loss of cross-topic synthesis, screening data and verified persistence.
func TestServiceLit(t *testing.T) {
	root := t.TempDir()
	a := session(t, root, "a", `{"title":"Scaling models","abstract":"Scaling models","identifiers":{"DOI":"10.1/shared"}}`)
	session(t, root, "b", `{"title":"Scaling models","abstract":"Scaling models","identifiers":{"DOI":"10.1/shared"}}`)
	if err := os.WriteFile(filepath.Join(a, "screening.jsonl"), []byte(`{"decision":"include"}`), 0644); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	got, err := (research.Service{}).Lit(research.LitOptions{ResearchDir: root, Out: out})
	if err != nil {
		t.Fatal(err)
	}
	if got.Topics != 2 || got.UniqueDOIs != 1 || got.CrossTopic != 1 || !got.HasScreening || got.KeywordThemes != 2 {
		t.Fatalf("lit = %+v", got)
	}
	data, err := os.ReadFile(filepath.Join(out, "LIT.md"))
	if err != nil || string(data) != got.Markdown || !strings.Contains(got.Markdown, "1 incl/0 excl") {
		t.Fatalf("synthesis artifact = %s, %v", data, err)
	}
	receipt, err := provenance.Validate(filepath.Join(out, "LIT.provenance.json"))
	if err != nil || receipt.Level != provenance.EngineVerified || got.RunID != receipt.RunID {
		t.Fatalf("receipt = %+v, %v", receipt, err)
	}
}

// Catches history fallback, matching/sorting/limits and persisted typed results.
func TestServiceHistory(t *testing.T) {
	root := t.TempDir()
	session(t, root, "a", `{"title":"Scaling","year":2020,"identifiers":{"DOI":"10.1/old"}}
{"title":"Other","abstract":"scaling","year":2026}`)
	b := session(t, root, "b", `{"Title":"Scaling","year":2026,"Identifiers":{"DOI":"10.1/new"}}`)
	if err := os.Rename(filepath.Join(b, "results-deduped.jsonl"), filepath.Join(b, "results.jsonl")); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "history.json")
	got, err := (research.Service{}).History(research.HistoryOptions{ResearchDir: root, Term: "SCALING", Limit: 1, Out: out})
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 1 || got.Results[0].DOI != "10.1/new" || got.Results[0].Matches != 4 {
		t.Fatalf("history = %+v", got)
	}
	data, err := os.ReadFile(out)
	if err != nil || !strings.Contains(string(data), `"doi": "10.1/new"`) {
		t.Fatalf("history artifact = %s, %v", data, err)
	}
}
