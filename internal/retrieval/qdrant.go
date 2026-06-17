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

const (
	PayloadPrivacyFull     = "full-text"
	PayloadPrivacyRedacted = "redacted-checksum"
)

// EmbeddingModel embeds passage/query text for vector retrieval.
type EmbeddingModel interface {
	EmbeddingBackendName() string
	Embed(string) ([]float64, error)
}

type EmbeddingProviderRegistry struct {
	SchemaVersion string              `json:"schemaVersion"`
	Providers     []EmbeddingProvider `json:"providers"`
}

type EmbeddingProvider struct {
	Name                         string                     `json:"name"`
	ModelID                      string                     `json:"modelId"`
	ProviderKind                 string                     `json:"providerKind"`
	Dimensions                   int                        `json:"dimensions"`
	LicenseNotes                 string                     `json:"licenseNotes"`
	VectorIndexInvalidation      string                     `json:"vectorIndexInvalidation"`
	RetrievalBenchmarkCompatible bool                       `json:"retrievalBenchmarkCompatible"`
	Compliance                   EmbeddingComplianceProfile `json:"compliance"`
}

type EmbeddingComplianceProfile struct {
	TextEgress       string `json:"textEgress"`
	RequiresConsent  bool   `json:"requiresConsent"`
	RequiredConfig   string `json:"requiredConfig"`
	ModelVersionLock string `json:"modelVersionLock"`
	Dimensionality   string `json:"dimensionality"`
	RetentionPolicy  string `json:"retentionPolicy"`
	Redaction        string `json:"redaction"`
}

func DefaultEmbeddingProviderRegistry() EmbeddingProviderRegistry {
	return EmbeddingProviderRegistry{SchemaVersion: "1", Providers: []EmbeddingProvider{
		{Name: "deterministic-hash", ModelID: "rforge-deterministic-hash", ProviderKind: "local", Dimensions: 8, LicenseNotes: "ResearchForge fixture scaffold; no external model license", VectorIndexInvalidation: "invalidate when RFORGE_EMBEDDING_DIMENSIONS changes", RetrievalBenchmarkCompatible: true, Compliance: EmbeddingComplianceProfile{TextEgress: "none", RequiresConsent: false, RequiredConfig: "none", ModelVersionLock: "RFORGE_EMBEDDING_DIMENSIONS recorded in retrieval lock", Dimensionality: "fixed by RFORGE_EMBEDDING_DIMENSIONS or default 8", RetentionPolicy: "local-only", Redaction: "not-required"}},
		{Name: "http-embedding", ModelID: "RFORGE_EMBEDDING_MODEL", ProviderKind: "remote-or-local-service", Dimensions: 0, LicenseNotes: "operator must record model license/terms for the configured SentenceTransformers-compatible service", VectorIndexInvalidation: "invalidate Qdrant collections when RFORGE_EMBEDDING_MODEL or dimensions change", RetrievalBenchmarkCompatible: true, Compliance: EmbeddingComplianceProfile{TextEgress: "passage/query text sent to RFORGE_EMBEDDING_URL", RequiresConsent: true, RequiredConfig: "RFORGE_EMBEDDING_URL, RFORGE_EMBEDDING_MODEL, RFORGE_EMBEDDING_CONSENT=1", ModelVersionLock: "RFORGE_EMBEDDING_MODEL recorded in retrieval lock", Dimensionality: "reported by provider response and locked in qdrant report", RetentionPolicy: "provider-configured; reviewer must confirm before use", Redaction: "caller-managed; use RFORGE_QDRANT_PAYLOAD_PRIVACY=redacted-checksum for stored payloads"}},
	}}
}

func ValidateEmbeddingProviderCompliance(providerName string, consent bool, config map[string]string) error {
	provider, ok := DefaultEmbeddingProviderRegistry().Provider(providerName)
	if !ok {
		return fmt.Errorf("unknown embedding provider %q", providerName)
	}
	if provider.Compliance.RequiresConsent && !consent {
		return fmt.Errorf("embedding provider %s requires explicit consent before text egress", provider.Name)
	}
	if provider.Name == "http-embedding" {
		if strings.TrimSpace(config["RFORGE_EMBEDDING_URL"]) == "" {
			return fmt.Errorf("RFORGE_EMBEDDING_URL is required for http embedding provider")
		}
		if strings.TrimSpace(config["RFORGE_EMBEDDING_MODEL"]) == "" {
			return fmt.Errorf("RFORGE_EMBEDDING_MODEL is required to lock http embedding model version")
		}
	}
	return nil
}

func (r EmbeddingProviderRegistry) Provider(name string) (EmbeddingProvider, bool) {
	for _, provider := range r.Providers {
		if provider.Name == name || strings.HasPrefix(name, provider.Name+":") {
			return provider, true
		}
	}
	return EmbeddingProvider{}, false
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
	baseURL                string
	collection             string
	client                 *http.Client
	embeddings             EmbeddingModel
	payloadPrivacy         string
	invalidateBeforeUpsert bool
}

// QdrantOptions configures the optional Qdrant adapter.
type QdrantOptions struct {
	BaseURL                string
	Collection             string
	Timeout                time.Duration
	Embeddings             EmbeddingModel
	PayloadPrivacy         string
	InvalidateBeforeUpsert bool
}

type QdrantRebuildReport struct {
	SchemaVersion           string `json:"schemaVersion"`
	Backend                 string `json:"backend"`
	Collection              string `json:"collection"`
	EmbeddingProvider       string `json:"embeddingProvider"`
	Dimension               int    `json:"dimension"`
	PayloadPrivacy          string `json:"payloadPrivacy"`
	InvalidatedBeforeUpsert bool   `json:"invalidatedBeforeUpsert"`
	Attempted               int    `json:"attempted"`
	Indexed                 int    `json:"indexed"`
	TextEgress              string `json:"textEgress"`
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
	privacy := strings.TrimSpace(options.PayloadPrivacy)
	if privacy == "" {
		privacy = PayloadPrivacyFull
	}
	return &QdrantIndex{baseURL: baseURL, collection: collection, client: &http.Client{Timeout: timeout}, embeddings: embeddings, payloadPrivacy: privacy, invalidateBeforeUpsert: options.InvalidateBeforeUpsert}, nil
}

func (q *QdrantIndex) VectorBackendName() string { return "qdrant" }

// Rebuild upserts parsed passages into Qdrant with deterministic point IDs.
func (q *QdrantIndex) Rebuild(docs []parsing.ParsedDocument) error {
	_, err := q.RebuildWithReport(docs)
	return err
}

func (q *QdrantIndex) RebuildWithReport(docs []parsing.ParsedDocument) (QdrantRebuildReport, error) {
	providerName := q.embeddings.EmbeddingBackendName()
	report := QdrantRebuildReport{SchemaVersion: "1", Backend: "qdrant", Collection: q.collection, EmbeddingProvider: providerName, PayloadPrivacy: q.payloadPrivacy, InvalidatedBeforeUpsert: q.invalidateBeforeUpsert}
	points := []qdrantPoint{}
	for _, doc := range docs {
		for _, section := range doc.Sections {
			for _, passage := range section.Passages {
				vector, err := q.embeddings.Embed(passage.Text)
				if err != nil {
					return report, err
				}
				if report.Dimension == 0 {
					report.Dimension = len(vector)
				}
				points = append(points, qdrantPoint{ID: stablePointID(passage), Vector: vector, Payload: q.payloadForPassage(passage)})
				report.Attempted++
			}
		}
	}
	if report.Dimension == 0 {
		if probe, err := q.embeddings.Embed("dimension probe"); err == nil {
			report.Dimension = len(probe)
		}
	}
	if provider, ok := DefaultEmbeddingProviderRegistry().Provider(providerName); ok {
		report.TextEgress = provider.Compliance.TextEgress
	}
	if err := q.ensureCollection(context.Background(), report.Dimension); err != nil {
		return report, err
	}
	if q.invalidateBeforeUpsert {
		if err := q.invalidateCollection(context.Background()); err != nil {
			return report, err
		}
	}
	request := map[string]any{"points": points}
	body, _ := json.Marshal(request)
	if err := q.do(context.Background(), http.MethodPut, "/collections/"+q.collection+"/points?wait=true", body, nil); err != nil {
		return report, err
	}
	report.Indexed = len(points)
	return report, nil
}

func (q *QdrantIndex) payloadForPassage(passage parsing.Passage) any {
	if q.payloadPrivacy == PayloadPrivacyRedacted {
		sum := sha256.Sum256([]byte(passage.Text))
		return map[string]any{"PaperID": passage.PaperID, "SectionID": passage.SectionID, "PassageID": passage.ID, "TextChecksum": fmt.Sprintf("%x", sum[:]), "PayloadRedacted": true}
	}
	return PassageResult{PaperID: passage.PaperID, SectionID: passage.SectionID, PassageID: passage.ID, Text: passage.Text}
}

func (q *QdrantIndex) ensureCollection(ctx context.Context, dimension int) error {
	if dimension <= 0 {
		dimension = 8
	}
	request := map[string]any{"vectors": map[string]any{"size": dimension, "distance": "Cosine"}}
	body, _ := json.Marshal(request)
	requestHTTP, err := http.NewRequestWithContext(ctx, http.MethodPut, q.baseURL+"/collections/"+q.collection, bytes.NewReader(body))
	if err != nil {
		return err
	}
	requestHTTP.Header.Set("Content-Type", "application/json")
	response, err := q.client.Do(requestHTTP)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 10<<20))
	if err != nil {
		return err
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	if response.StatusCode == http.StatusBadRequest && strings.Contains(string(data), "already") {
		return nil
	}
	return fmt.Errorf("qdrant HTTP status %d: %s", response.StatusCode, strings.TrimSpace(string(data)))
}

func (q *QdrantIndex) invalidateCollection(ctx context.Context) error {
	request := map[string]any{"filter": map[string]any{}}
	body, _ := json.Marshal(request)
	return q.do(ctx, http.MethodPost, "/collections/"+q.collection+"/points/delete?wait=true", body, nil)
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
	ID      string    `json:"id"`
	Vector  []float64 `json:"vector"`
	Payload any       `json:"payload"`
}

type qdrantSearchResponse struct {
	Result []struct {
		Payload PassageResult `json:"payload"`
	} `json:"result"`
}
