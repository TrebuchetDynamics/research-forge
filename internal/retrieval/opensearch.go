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

const OpenSearchMappingVersion = "opensearch-passages-v1"

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

type OpenSearchBulkReport struct {
	SchemaVersion  string                  `json:"schemaVersion"`
	Backend        string                  `json:"backend"`
	Index          string                  `json:"index"`
	MappingVersion string                  `json:"mappingVersion"`
	Attempted      int                     `json:"attempted"`
	Indexed        int                     `json:"indexed"`
	Failed         int                     `json:"failed"`
	Failures       []OpenSearchBulkFailure `json:"failures,omitempty"`
}

type OpenSearchBulkFailure struct {
	DocumentID string `json:"documentId"`
	Status     int    `json:"status"`
	Type       string `json:"type,omitempty"`
	Reason     string `json:"reason,omitempty"`
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
	_, err := o.RebuildWithReport(docs)
	return err
}

// RebuildWithReport bulk-indexes parsed passages and records partial-failure provenance.
func (o *OpenSearchIndex) RebuildWithReport(docs []parsing.ParsedDocument) (OpenSearchBulkReport, error) {
	report := OpenSearchBulkReport{SchemaVersion: "1", Backend: "opensearch", Index: o.index, MappingVersion: OpenSearchMappingVersion}
	if err := o.ensureMapping(context.Background()); err != nil {
		return report, err
	}
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
				report.Attempted++
			}
		}
	}
	if bulk.Len() == 0 {
		bulk.WriteString("\n")
	}
	var response openSearchBulkResponse
	if err := o.do(context.Background(), http.MethodPost, "/"+o.index+"/_bulk", "application/x-ndjson", []byte(bulk.String()), &response); err != nil {
		return report, err
	}
	report.Indexed = report.Attempted
	for _, item := range response.Items {
		entry := item.Index
		if entry.Status >= 300 || entry.Error.Reason != "" || entry.Error.Type != "" {
			report.Failures = append(report.Failures, OpenSearchBulkFailure{DocumentID: entry.ID, Status: entry.Status, Type: entry.Error.Type, Reason: entry.Error.Reason})
		}
	}
	report.Failed = len(report.Failures)
	report.Indexed = report.Attempted - report.Failed
	if err := o.do(context.Background(), http.MethodPost, "/"+o.index+"/_refresh", "application/json", nil, nil); err != nil {
		return report, err
	}
	return report, nil
}

func (o *OpenSearchIndex) ensureMapping(ctx context.Context) error {
	mapping := map[string]any{
		"settings": map[string]any{"index": map[string]any{"mapping_version": OpenSearchMappingVersion}},
		"mappings": map[string]any{"properties": map[string]any{
			"PaperID":   map[string]any{"type": "keyword"},
			"SectionID": map[string]any{"type": "keyword"},
			"PassageID": map[string]any{"type": "keyword"},
			"Text":      map[string]any{"type": "text"},
		}},
	}
	body, _ := json.Marshal(mapping)
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, o.baseURL+"/"+o.index, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := o.client.Do(request)
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
	if response.StatusCode == http.StatusBadRequest && strings.Contains(string(data), "resource_already_exists_exception") {
		return nil
	}
	return fmt.Errorf("opensearch HTTP status %d: %s", response.StatusCode, strings.TrimSpace(string(data)))
}

// Retrieve searches OpenSearch for matching passages.
func (o *OpenSearchIndex) Retrieve(query string) ([]PassageResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("retrieve query is required")
	}
	request := map[string]any{"size": 20, "query": map[string]any{"match": map[string]any{"Text": query}}, "highlight": map[string]any{"fields": map[string]any{"Text": map[string]any{}}}}
	body, _ := json.Marshal(request)
	var response openSearchResponse
	if err := o.do(context.Background(), http.MethodPost, "/"+o.index+"/_search", "application/json", body, &response); err != nil {
		return nil, err
	}
	results := make([]PassageResult, 0, len(response.Hits.Hits))
	for _, hit := range response.Hits.Hits {
		result := hit.Source
		result.Highlights = append(result.Highlights, hit.Highlight.Text...)
		results = append(results, result)
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

type openSearchBulkResponse struct {
	Errors bool `json:"errors"`
	Items  []struct {
		Index struct {
			ID     string `json:"_id"`
			Status int    `json:"status"`
			Error  struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
			} `json:"error"`
		} `json:"index"`
	} `json:"items"`
}

type openSearchResponse struct {
	Hits struct {
		Hits []struct {
			Source    PassageResult `json:"_source"`
			Highlight struct {
				Text []string `json:"Text"`
			} `json:"highlight"`
		} `json:"hits"`
	} `json:"hits"`
}
