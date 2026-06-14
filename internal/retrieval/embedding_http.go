package retrieval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPEmbedding calls an opt-in local embedding service for passage/query vectors.
type HTTPEmbedding struct {
	Endpoint string
	Model    string
	Client   *http.Client
}

func (h HTTPEmbedding) EmbeddingBackendName() string {
	if strings.TrimSpace(h.Model) != "" {
		return "http-embedding:" + strings.TrimSpace(h.Model)
	}
	return "http-embedding"
}

func (h HTTPEmbedding) Embed(text string) ([]float64, error) {
	endpoint := strings.TrimSpace(h.Endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("embedding endpoint is required")
	}
	client := h.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	payload := map[string]string{"text": text}
	if strings.TrimSpace(h.Model) != "" {
		payload["model"] = strings.TrimSpace(h.Model)
	}
	body, _ := json.Marshal(payload)
	request, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 10<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding HTTP status %d: %s", response.StatusCode, strings.TrimSpace(string(data)))
	}
	var decoded struct {
		Embedding []float64 `json:"embedding"`
		Vector    []float64 `json:"vector"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	vector := decoded.Embedding
	if len(vector) == 0 {
		vector = decoded.Vector
	}
	if len(vector) == 0 {
		return nil, fmt.Errorf("embedding response missing vector")
	}
	return vector, nil
}
