package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FigshareConnector searches the figshare research outputs repository.
type FigshareConnector struct {
	http HTTPClient
}

// NewFigshareConnector creates a figshare source connector.
func NewFigshareConnector(httpClient HTTPClient) FigshareConnector {
	return FigshareConnector{http: httpClient}
}

// Name returns the connector source name.
func (FigshareConnector) Name() string { return "figshare" }

// Search queries the figshare API and normalizes results into SourceRecords.
func (c FigshareConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"search_for":      query.Terms,
		"page_size":       strconv.Itoa(limit),
		"order":           "published_date",
		"order_direction": "desc",
	}
	body, err := c.http.Get(ctx, "/v2/articles", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var items []figshareArticle
	if err := json.Unmarshal(body, &items); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(items))
	for _, item := range items {
		sourceID := strconv.FormatInt(item.ID, 10)
		doi := normalizeSourceDOI(strings.TrimSpace(item.DOI))
		crossrefID := ""
		if doi == "" {
			crossrefID = "figshare:" + sourceID
		}

		year := 0
		if len(item.PublishedDate) >= 4 {
			year, _ = strconv.Atoi(item.PublishedDate[:4])
		}

		licenseName := ""
		openAccess := false
		if item.License != nil {
			licenseName = strings.TrimSpace(item.License.Name)
			if strings.Contains(item.License.URL, "creativecommons.org") {
				openAccess = true
			}
		}

		records = append(records, SourceRecord{
			Source:   "figshare",
			SourceID: sourceID,
			Title:    strings.TrimSpace(item.Title),
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:      year,
			Abstract:  strings.TrimSpace(item.Description),
			Venue:     strings.TrimSpace(item.DefinedTypeName),
			URLs:      nonEmptyStrings(strings.TrimSpace(item.URLPublicHTML)),
			License:   licenseName,
			OpenAccess: openAccess,
			Metadata: map[string]string{
				"defined_type": strconv.Itoa(item.DefinedType),
				"type_name":    strings.TrimSpace(item.DefinedTypeName),
			},
		})
	}
	return SourceResponse{
		Records: records,
		RawRef:  fmt.Sprintf("figshare:/v2/articles?search_for=%s&page_size=%d", query.Terms, limit),
	}, nil
}

type figshareArticle struct {
	ID              int64            `json:"id"`
	Title           string           `json:"title"`
	DOI             string           `json:"doi"`
	URL             string           `json:"url"`
	URLPublicHTML   string           `json:"url_public_html"`
	PublishedDate   string           `json:"published_date"`
	DefinedType     int              `json:"defined_type"`
	DefinedTypeName string           `json:"defined_type_name"`
	Description     string           `json:"description"`
	License         *figshareLicense `json:"license"`
}

type figshareLicense struct {
	Value int    `json:"value"`
	Name  string `json:"name"`
	URL   string `json:"url"`
}
