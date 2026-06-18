package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// LensConnector searches Lens.org scholarly and patent literature.
type LensConnector struct {
	http HTTPClient
}

// NewLensConnector creates a Lens.org source connector.
func NewLensConnector(httpClient HTTPClient) LensConnector {
	return LensConnector{http: httpClient}
}

// Name returns the connector source name.
func (LensConnector) Name() string { return "lens" }

// Search queries the Lens.org scholarly search API and normalizes results into SourceRecords.
func (c LensConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}

	requestBody := lensSearchRequest{
		Query: lensQuery{
			QueryString: lensQueryString{
				Query:  query.Terms,
				Fields: []string{"title", "abstract"},
			},
		},
		Size: limit,
		Include: []string{
			"lens_id",
			"title",
			"abstract",
			"year_published",
			"authors",
			"open_access",
			"external_ids",
			"source",
			"scholarly_citations_count",
		},
		Sort: []map[string]string{{"year_published": "desc"}},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return SourceResponse{}, err
	}

	respBytes, err := c.http.Post(ctx, "/scholarly/search", bodyBytes)
	if err != nil {
		return SourceResponse{}, err
	}

	var payload lensSearchResponse
	if err := json.Unmarshal(respBytes, &payload); err != nil {
		return SourceResponse{}, err
	}

	records := make([]SourceRecord, 0, len(payload.Data))
	for _, item := range payload.Data {
		lensID := strings.TrimSpace(item.LensID)

		doi := ""
		arxivID := ""
		for _, extID := range item.ExternalIDs {
			switch extID.Type {
			case "doi":
				if doi == "" {
					doi = normalizeSourceDOI(extID.Value)
				}
			case "arxiv":
				if arxivID == "" {
					arxivID = strings.TrimSpace(extID.Value)
				}
			}
		}

		crossrefID := ""
		if doi == "" && arxivID == "" {
			crossrefID = "lens:" + lensID
		}

		records = append(records, SourceRecord{
			Source:   "lens",
			SourceID: lensID,
			Title:    strings.TrimSpace(item.Title),
			Identifiers: Identifiers{
				DOI:        doi,
				ArXivID:    arxivID,
				CrossrefID: crossrefID,
			},
			Year:       item.YearPublished,
			Abstract:   strings.TrimSpace(item.Abstract),
			Venue:      strings.TrimSpace(item.Source.Title),
			Publisher:  strings.TrimSpace(item.Source.Publisher),
			OpenAccess: item.OpenAccess.IsOA,
			URLs:       []string{},
			Metadata: map[string]string{
				"lens_id":   lensID,
				"citations": strconv.Itoa(item.ScholarlycitationsCount),
			},
		})
	}
	return SourceResponse{
		Records: records,
		RawRef:  fmt.Sprintf("lens:/scholarly/search?q=%s&size=%d", query.Terms, limit),
	}, nil
}

type lensSearchRequest struct {
	Query   lensQuery           `json:"query"`
	Size    int                 `json:"size"`
	Include []string            `json:"include"`
	Sort    []map[string]string `json:"sort"`
}

type lensQuery struct {
	QueryString lensQueryString `json:"query_string"`
}

type lensQueryString struct {
	Query  string   `json:"query"`
	Fields []string `json:"fields"`
}

type lensSearchResponse struct {
	Total int          `json:"total"`
	Data  []lensRecord `json:"data"`
}

type lensRecord struct {
	LensID                  string           `json:"lens_id"`
	Title                   string           `json:"title"`
	Abstract                string           `json:"abstract"`
	YearPublished           int              `json:"year_published"`
	ScholarlycitationsCount int              `json:"scholarly_citations_count"`
	Authors                 []lensAuthor     `json:"authors"`
	OpenAccess              lensOpenAccess   `json:"open_access"`
	ExternalIDs             []lensExternalID `json:"external_ids"`
	Source                  lensSource       `json:"source"`
}

type lensAuthor struct {
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	IDs       []lensID `json:"ids"`
}

type lensID struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type lensOpenAccess struct {
	IsOA  bool   `json:"is_oa"`
	Color string `json:"color"`
}

type lensExternalID struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type lensSource struct {
	Title     string `json:"title"`
	Publisher string `json:"publisher"`
}
