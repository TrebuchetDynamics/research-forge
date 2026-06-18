package sources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHuggingFaceSearchNormalizesRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/papers" {
			t.Fatalf("path = %q, want /api/papers", r.URL.Path)
		}
		if r.URL.Query().Get("q") == "" {
			t.Fatal("missing q param")
		}
		_, _ = w.Write([]byte(`[{
			"id": "2303.08774",
			"title": "GPT-4 Technical Report",
			"publishedAt": "2023-03-15T00:00:00.000Z",
			"authors": [
				{"name": "OpenAI"},
				{"name": "Josh Achiam"}
			],
			"summary": "We report the development of GPT-4, a large-scale, multimodal model.",
			"upvotes": 1234,
			"githubRepo": ""
		}]`))
	}))
	defer server.Close()

	connector := NewHuggingFaceConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "gpt-4 technical report", Limit: 5})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Source != "huggingface" {
		t.Fatalf("Source = %q, want huggingface", r.Source)
	}
	if r.SourceID != "2303.08774" {
		t.Fatalf("SourceID = %q, want 2303.08774", r.SourceID)
	}
	// DOI is the normalized arXiv DOI (lowercase arXiv).
	if r.Identifiers.DOI != "10.48550/arxiv.2303.08774" {
		t.Fatalf("DOI = %q, want 10.48550/arxiv.2303.08774", r.Identifiers.DOI)
	}
	if r.Identifiers.CrossrefID != "" {
		t.Fatalf("CrossrefID = %q, want empty when DOI set", r.Identifiers.CrossrefID)
	}
	if r.Title != "GPT-4 Technical Report" {
		t.Fatalf("Title = %q", r.Title)
	}
	if r.Year != 2023 {
		t.Fatalf("Year = %d, want 2023", r.Year)
	}
	if r.Abstract != "We report the development of GPT-4, a large-scale, multimodal model." {
		t.Fatalf("Abstract = %q", r.Abstract)
	}
	if !r.OpenAccess {
		t.Fatal("OpenAccess = false, want true")
	}
	if len(r.URLs) != 1 || r.URLs[0] != "https://arxiv.org/abs/2303.08774" {
		t.Fatalf("URLs = %v", r.URLs)
	}
	if r.Metadata["authors"] != "OpenAI; Josh Achiam" {
		t.Fatalf("authors = %q, want OpenAI; Josh Achiam", r.Metadata["authors"])
	}
	if r.Metadata["upvotes"] != "1234" {
		t.Fatalf("upvotes = %q, want 1234", r.Metadata["upvotes"])
	}

	papers, err := PaperRecords(response)
	if err != nil {
		t.Fatalf("PaperRecords error: %v", err)
	}
	if len(papers) != 1 || papers[0].Title != "GPT-4 Technical Report" {
		t.Fatalf("papers round-trip failed")
	}
}

func TestHuggingFaceSearchWithGitHubRepo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{
			"id": "2310.06825",
			"title": "Mistral 7B",
			"publishedAt": "2023-10-10T00:00:00.000Z",
			"authors": [{"name": "Albert Jiang"}],
			"summary": "We introduce Mistral 7B v0.1.",
			"upvotes": 5678,
			"githubRepo": "https://github.com/mistralai/mistral-src"
		}]`))
	}))
	defer server.Close()

	connector := NewHuggingFaceConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "mistral"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(response.Records))
	}
	r := response.Records[0]
	if r.Metadata["github_repo"] != "https://github.com/mistralai/mistral-src" {
		t.Fatalf("github_repo = %q", r.Metadata["github_repo"])
	}
	if r.Identifiers.DOI != "10.48550/arxiv.2310.06825" {
		t.Fatalf("DOI = %q, want 10.48550/arxiv.2310.06825", r.Identifiers.DOI)
	}
}

func TestHuggingFaceSearchSkipsBlankTitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"id": "2401.00001", "title": "", "publishedAt": "2024-01-01T00:00:00.000Z", "authors": [], "summary": "empty title", "upvotes": 5, "githubRepo": ""},
			{"id": "2401.00002", "title": "Valid HuggingFace Paper", "publishedAt": "2024-01-02T00:00:00.000Z", "authors": [], "summary": "valid abstract", "upvotes": 10, "githubRepo": ""}
		]`))
	}))
	defer server.Close()

	connector := NewHuggingFaceConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	response, err := connector.Search(context.Background(), SourceQuery{Terms: "test"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(response.Records) != 1 {
		t.Fatalf("records = %d, want 1 (blank title skipped)", len(response.Records))
	}
	if response.Records[0].Title != "Valid HuggingFace Paper" {
		t.Fatalf("Title = %q, want Valid HuggingFace Paper", response.Records[0].Title)
	}
}

func TestHuggingFaceSearchDefaultLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "25" {
			t.Fatalf("default limit = %q, want 25", r.URL.Query().Get("limit"))
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	connector := NewHuggingFaceConnector(NewHTTPClient(HTTPClientOptions{BaseURL: server.URL, Timeout: time.Second}))
	_, err := connector.Search(context.Background(), SourceQuery{Terms: "diffusion model", Limit: 0})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
}
