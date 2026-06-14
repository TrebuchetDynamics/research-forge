package retrieval

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

// EmbeddingModel embeds passage/query text for vector retrieval.
type EmbeddingModel interface {
	EmbeddingBackendName() string
	Embed(string) ([]float64, error)
}

// DeterministicEmbedding is a local, dependency-free embedding scaffold for tests and offline workflows.
type DeterministicEmbedding struct{ Dimensions int }

func (d DeterministicEmbedding) EmbeddingBackendName() string { return "deterministic-hash" }

func (d DeterministicEmbedding) Embed(text string) ([]float64, error) {
	dims := d.Dimensions
	if dims <= 0 {
		dims = 8
	}
	vector := make([]float64, dims)
	for _, token := range strings.Fields(strings.ToLower(text)) {
		sum := sha256.Sum256([]byte(token))
		idx := int(sum[0]) % dims
		vector[idx] += 1
	}
	norm := 0.0
	for _, value := range vector {
		norm += value * value
	}
	if norm == 0 {
		return vector, nil
	}
	norm = math.Sqrt(norm)
	for i := range vector {
		vector[i] /= norm
	}
	return vector, nil
}

// QdrantIndex is an optional passage vector retrieval adapter backed by Qdrant.
type QdrantIndex struct {
	baseURL    string
	collection string
	client     *http.Client
	embeddings EmbeddingModel
}

// QdrantOptions configures the optional Qdrant adapter.
type QdrantOptions struct {
	BaseURL    string
	Collection string
	Timeout    time.Duration
	Embeddings EmbeddingModel
}

func NewQdrantIndex(options QdrantOptions) (*QdrantIndex, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(options.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("qdrant base URL is required")
	}
	collection := strings.Trim(strings.TrimSpace(options.Collection), "/")
	if collection == "" {
		collection = "researchforge_passages"
	}
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	embeddings := options.Embeddings
	if embeddings == nil {
		embeddings = DeterministicEmbedding{Dimensions: 8}
	}
	return &QdrantIndex{baseURL: baseURL, collection: collection, client: &http.Client{Timeout: timeout}, embeddings: embeddings}, nil
}

func (q *QdrantIndex) VectorBackendName() string { return "qdrant" }

// Rebuild upserts parsed passages into Qdrant with deterministic point IDs.
func (q *QdrantIndex) Rebuild(docs []parsing.ParsedDocument) error {
	points := []qdrantPoint{}
	for _, doc := range docs {
		for _, section := range doc.Sections {
			for _, passage := range section.Passages {
				vector, err := q.embeddings.Embed(passage.Text)
				if err != nil {
					return err
				}
				points = append(points, qdrantPoint{ID: stablePointID(passage), Vector: vector, Payload: PassageResult{PaperID: passage.PaperID, SectionID: passage.SectionID, PassageID: passage.ID, Text: passage.Text}})
			}
		}
	}
	request := map[string]any{"points": points}
	body, _ := json.Marshal(request)
	return q.do(context.Background(), http.MethodPut, "/collections/"+q.collection+"/points?wait=true", body, nil)
}

// Retrieve searches Qdrant by embedded query vector.
func (q *QdrantIndex) Retrieve(query string) ([]PassageResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("retrieve query is required")
	}
	vector, err := q.embeddings.Embed(query)
	if err != nil {
		return nil, err
	}
	request := map[string]any{"vector": vector, "limit": 20, "with_payload": true}
	body, _ := json.Marshal(request)
	var response qdrantSearchResponse
	if err := q.do(context.Background(), http.MethodPost, "/collections/"+q.collection+"/points/search", body, &response); err != nil {
		return nil, err
	}
	results := make([]PassageResult, 0, len(response.Result))
	for _, hit := range response.Result {
		results = append(results, hit.Payload)
	}
	return results, nil
}

func (q *QdrantIndex) Close() error { return nil }

func (q *QdrantIndex) do(ctx context.Context, method, path string, body []byte, out any) error {
	request, err := http.NewRequestWithContext(ctx, method, q.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := q.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 10<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("qdrant HTTP status %d: %s", response.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return err
		}
	}
	return nil
}

func stablePointID(passage parsing.Passage) string {
	id := passage.ID
	if id == "" {
		id = passage.PaperID + ":" + passage.SectionID + ":" + passage.Text
	}
	sum := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%x", sum[:16])
}

type qdrantPoint struct {
	ID      string        `json:"id"`
	Vector  []float64     `json:"vector"`
	Payload PassageResult `json:"payload"`
}

type qdrantSearchResponse struct {
	Result []struct {
		Payload PassageResult `json:"payload"`
	} `json:"result"`
}
