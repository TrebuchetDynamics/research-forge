package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// NTRSConnector searches the NASA Technical Reports Server (NTRS).
type NTRSConnector struct {
	http HTTPClient
}

// NewNTRSConnector creates an NTRS source connector.
func NewNTRSConnector(httpClient HTTPClient) NTRSConnector {
	return NTRSConnector{http: httpClient}
}

// Name returns the connector source name.
func (NTRSConnector) Name() string { return "ntrs" }

// Search queries the NTRS REST API and normalizes results into SourceRecords.
func (c NTRSConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"keyword": query.Terms,
		"count":   strconv.Itoa(limit),
	}
	body, err := c.http.Get(ctx, "/api/citations/search", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload ntrsSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Results))
	for _, result := range payload.Results {
		title := strings.TrimSpace(result.Title)
		if title == "" {
			continue
		}
		idStr := strconv.Itoa(result.ID)
		crossrefID := "ntrs:" + idStr
		htmlURL := fmt.Sprintf("https://ntrs.nasa.gov/citations/%d", result.ID)
		year := 0
		dateStr := strings.TrimSpace(result.DistributionDate)
		if len(dateStr) >= 4 {
			year, _ = strconv.Atoi(dateStr[:4])
		}
		if year == 0 {
			dateStr = strings.TrimSpace(result.SubmittedDate)
			if len(dateStr) >= 4 {
				year, _ = strconv.Atoi(dateStr[:4])
			}
		}
		venue := strings.TrimSpace(result.Center.Name)
		keywords := strings.Join(result.Keywords, "; ")
		records = append(records, SourceRecord{
			Source:   "ntrs",
			SourceID: idStr,
			Title:    title,
			Identifiers: Identifiers{
				CrossrefID: crossrefID,
			},
			Year:       year,
			Abstract:   strings.TrimSpace(result.Abstract),
			Venue:      venue,
			URLs:       nonEmptyStrings(htmlURL),
			OpenAccess: true,
			Metadata: map[string]string{
				"stiType":     strings.TrimSpace(result.STIType),
				"center_code": strings.TrimSpace(result.Center.Code),
				"keywords":    keywords,
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawNTRSRef(query.Terms, limit)}, nil
}

type ntrsSearchResponse struct {
	Results []ntrsResult `json:"results"`
}

type ntrsResult struct {
	ID               int      `json:"id"`
	Title            string   `json:"title"`
	Abstract         string   `json:"abstract"`
	SubmittedDate    string   `json:"submittedDate"`
	DistributionDate string   `json:"distributionDate"`
	Keywords         []string `json:"keywords"`
	Center           struct {
		Code string `json:"code"`
		Name string `json:"name"`
	} `json:"center"`
	STIType string `json:"stiType"`
}

func rawNTRSRef(keyword string, count int) string {
	return fmt.Sprintf("ntrs:/api/citations/search?count=%d&keyword=%s",
		count, url.QueryEscape(keyword))
}
