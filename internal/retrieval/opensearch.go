package retrieval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
)

// OpenSearchIndex is an optional passage search adapter backed by OpenSearch.
type OpenSearchIndex struct {
	baseURL string
	index   string
	client  *http.Client
}

// OpenSearchOptions configures the optional OpenSearch adapter.
type OpenSearchOptions struct {
	BaseURL string
	Index   string
	Timeout time.Duration
}

// NewOpenSearchIndex creates an OpenSearch passage index adapter.
func NewOpenSearchIndex(options OpenSearchOptions) (*OpenSearchIndex, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(options.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("opensearch base URL is required")
	}
	index := strings.Trim(strings.TrimSpace(options.Index), "/")
	if index == "" {
		index = "researchforge-passages"
	}
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &OpenSearchIndex{baseURL: baseURL, index: index, client: &http.Client{Timeout: timeout}}, nil
}

// Rebuild bulk-indexes parsed passages into OpenSearch.
func (o *OpenSearchIndex) Rebuild(docs []parsing.ParsedDocument) error {
	var bulk strings.Builder
	for _, doc := range docs {
		for _, section := range doc.Sections {
			for _, passage := range section.Passages {
				id := passage.ID
				if id == "" {
					id = passage.PaperID + ":" + passage.SectionID
				}
				meta, _ := json.Marshal(map[string]any{"index": map[string]any{"_id": id}})
				docLine, _ := json.Marshal(PassageResult{PaperID: passage.PaperID, SectionID: passage.SectionID, PassageID: passage.ID, Text: passage.Text})
				bulk.Write(meta)
				bulk.WriteByte('\n')
				bulk.Write(docLine)
				bulk.WriteByte('\n')
			}
		}
	}
	if bulk.Len() == 0 {
		bulk.WriteString("\n")
	}
	if err := o.do(context.Background(), http.MethodPost, "/"+o.index+"/_bulk", "application/x-ndjson", []byte(bulk.String()), nil); err != nil {
		return err
	}
	return o.do(context.Background(), http.MethodPost, "/"+o.index+"/_refresh", "application/json", nil, nil)
}

// Retrieve searches OpenSearch for matching passages.
func (o *OpenSearchIndex) Retrieve(query string) ([]PassageResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("retrieve query is required")
	}
	request := map[string]any{"size": 20, "query": map[string]any{"match": map[string]any{"Text": query}}}
	body, _ := json.Marshal(request)
	var response openSearchResponse
	if err := o.do(context.Background(), http.MethodPost, "/"+o.index+"/_search", "application/json", body, &response); err != nil {
		return nil, err
	}
	results := make([]PassageResult, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		results = append(results, hit.Source)
	}
	return results, nil
}

// Close closes adapter resources. The HTTP client has no persistent resource to close.
func (o *OpenSearchIndex) Close() error { return nil }

func (o *OpenSearchIndex) do(ctx context.Context, method, path, contentType string, body []byte, out any) error {
	request, err := http.NewRequestWithContext(ctx, method, o.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := o.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 10<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("opensearch HTTP status %d: %s", response.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return err
		}
	}
	return nil
}

type openSearchResponse struct {
	Hits struct {
		Hits []struct {
			Source PassageResult `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}
