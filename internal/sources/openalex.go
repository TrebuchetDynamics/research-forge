package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// OpenAlexConnector searches OpenAlex works.
type OpenAlexConnector struct {
	http HTTPClient
}

// NewOpenAlexConnector creates an OpenAlex source connector.
func NewOpenAlexConnector(httpClient HTTPClient) OpenAlexConnector {
	return OpenAlexConnector{http: httpClient}
}

// Name returns the connector source name.
func (OpenAlexConnector) Name() string { return "openalex" }

// Search queries OpenAlex works and normalizes results into SourceRecords.
func (c OpenAlexConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{"search": query.Terms, "per-page": strconv.Itoa(limit)}
	if strings.TrimSpace(query.PageCursor) != "" {
		params["cursor"] = strings.TrimSpace(query.PageCursor)
	}
	body, err := c.http.Get(ctx, "/works", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload openAlexWorksResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Results))
	for _, work := range payload.Results {
		openAlexID := normalizeOpenAlexID(work.ID)
		records = append(records, SourceRecord{
			Source:   "openalex",
			SourceID: openAlexID,
			Title:    strings.TrimSpace(work.Title),
			Identifiers: Identifiers{
				DOI:        normalizeSourceDOI(work.DOI),
				OpenAlexID: openAlexID,
			},
			Year:       work.PublicationYear,
			URLs:       nonEmptyStrings(work.PrimaryLocation.LandingPageURL),
			License:    strings.TrimSpace(work.PrimaryLocation.License),
			OpenAccess: work.OpenAccess.IsOA,
			Metadata: map[string]string{
				"type":      strings.TrimSpace(work.Type),
				"oa_status": strings.TrimSpace(work.OpenAccess.OAStatus),
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawOpenAlexRef(params), NextPageCursor: strings.TrimSpace(payload.Meta.NextCursor)}, nil
}

type openAlexWorksResponse struct {
	Meta    openAlexMeta   `json:"meta"`
	Results []openAlexWork `json:"results"`
}

type openAlexMeta struct {
	NextCursor string `json:"next_cursor"`
}

type openAlexWork struct {
	ID              string                  `json:"id"`
	DOI             string                  `json:"doi"`
	Title           string                  `json:"title"`
	PublicationYear int                     `json:"publication_year"`
	Type            string                  `json:"type"`
	OpenAccess      openAlexOpenAccess      `json:"open_access"`
	PrimaryLocation openAlexPrimaryLocation `json:"primary_location"`
}

type openAlexOpenAccess struct {
	IsOA     bool   `json:"is_oa"`
	OAStatus string `json:"oa_status"`
}

type openAlexPrimaryLocation struct {
	LandingPageURL string `json:"landing_page_url"`
	License        string `json:"license"`
}

func rawOpenAlexRef(params map[string]string) string {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return fmt.Sprintf("openalex:/works?%s", values.Encode())
}

func nonEmptyStrings(values ...string) []string {
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func normalizeOpenAlexID(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "https://openalex.org/")
	return value
}

func normalizeSourceDOI(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "https://doi.org/")
	value = strings.TrimPrefix(value, "http://doi.org/")
	value = strings.TrimPrefix(value, "doi:")
	return strings.TrimSpace(value)
}
