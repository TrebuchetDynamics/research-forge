package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// CiNiiConnector searches the CiNii Research database.
//
// CiNii Research (cir.nii.ac.jp) is operated by the National Institute of
// Informatics (NII) of Japan. It aggregates articles, books, dissertations, and
// research data from Japanese and international scholarly sources. The OpenSearch
// API requires no authentication and returns JSON-LD.
type CiNiiConnector struct {
	http HTTPClient
}

// NewCiNiiConnector creates a CiNii source connector.
func NewCiNiiConnector(httpClient HTTPClient) CiNiiConnector {
	return CiNiiConnector{http: httpClient}
}

// Name returns the connector source name.
func (CiNiiConnector) Name() string { return "cinii" }

// Search queries the CiNii Research articles OpenSearch endpoint and normalizes
// results into SourceRecords.
func (c CiNiiConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"q":      query.Terms,
		"count":  strconv.Itoa(limit),
		"lang":   "en",
		"format": "json",
	}
	body, err := c.http.Get(ctx, "/opensearch/articles", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload ciniiSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Items))
	for _, item := range payload.Items {
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}

		doi := ""
		naid := ""
		for _, id := range item.Identifiers {
			switch id.Type {
			case "cir:DOI":
				doi = normalizeSourceDOI(id.Value)
			case "cir:NAID":
				naid = id.Value
			}
		}

		// CrossrefID fallback: prefer NAID, then CRID from the @id URL.
		crossrefID := ""
		if doi == "" {
			if naid != "" {
				crossrefID = "cinii-naid:" + naid
			} else {
				parts := strings.Split(strings.TrimRight(item.ID, "/"), "/")
				if len(parts) > 0 {
					crossrefID = "cinii:" + parts[len(parts)-1]
				}
			}
		}

		year := 0
		if len(item.PubDate) >= 4 {
			year, _ = strconv.Atoi(item.PubDate[:4])
		}

		abstract := stripHTMLTags(item.Description)

		authors := extractCiNiiCreators(item.Creator)
		metadata := map[string]string{}
		if len(authors) > 0 {
			metadata["authors"] = strings.Join(authors, "; ")
		}

		records = append(records, SourceRecord{
			Source:   "cinii",
			SourceID: item.ID,
			Title:    title,
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:      year,
			Abstract:  abstract,
			Venue:     item.PublicationName,
			Publisher: item.Publisher,
			URLs:      nonEmptyStrings(item.ID),
			Metadata:  metadata,
		})
	}
	rawRef := fmt.Sprintf("cinii:/opensearch/articles?q=%s&count=%d&lang=en&format=json", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

// extractCiNiiCreators handles dc:creator encoded as either a JSON string or
// a JSON array of strings.
func extractCiNiiCreators(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var arr []string
	if json.Unmarshal(raw, &arr) == nil {
		return arr
	}
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return []string{s}
	}
	return nil
}

// ciniiSearchResponse is the top-level JSON-LD response from the CiNii articles
// OpenSearch endpoint.
type ciniiSearchResponse struct {
	TotalResults int         `json:"opensearch:totalResults"`
	Items        []ciniiItem `json:"items"`
}

// ciniiItem is a single CiNii article entry in JSON-LD format.
type ciniiItem struct {
	ID              string            `json:"@id"`
	Title           string            `json:"title"`
	Creator         json.RawMessage   `json:"dc:creator"`
	Publisher       string            `json:"dc:publisher"`
	PublicationName string            `json:"prism:publicationName"`
	PubDate         string            `json:"prism:publicationDate"`
	Description     string            `json:"description"`
	Identifiers     []ciniiIdentifier `json:"dc:identifier"`
}

// ciniiIdentifier is a typed identifier entry in a CiNii item.
type ciniiIdentifier struct {
	Type  string `json:"@type"`
	Value string `json:"@value"`
}
