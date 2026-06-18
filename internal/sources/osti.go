package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// OSTIConnector searches the OSTI.gov (US DOE Office of Scientific and Technical Information) API.
type OSTIConnector struct {
	http HTTPClient
}

// NewOSTIConnector creates an OSTI source connector.
func NewOSTIConnector(httpClient HTTPClient) OSTIConnector {
	return OSTIConnector{http: httpClient}
}

// Name returns the connector source name.
func (OSTIConnector) Name() string { return "osti" }

// Search queries the OSTI REST API and normalizes results into SourceRecords.
func (c OSTIConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"q":    query.Terms,
		"page": "0",
		"size": strconv.Itoa(limit),
	}
	body, err := c.http.Get(ctx, "/api/v1/records", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var results []ostiRecord
	if err := json.Unmarshal(body, &results); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(results))
	for _, result := range results {
		title := strings.TrimSpace(result.Title)
		if title == "" {
			continue
		}
		idStr := strconv.Itoa(result.OstiID)
		doi := normalizeSourceDOI(result.DOI)

		crossrefID := ""
		if doi == "" {
			crossrefID = "osti:" + idStr
		}

		year := ""
		if len(result.PublicationDate) >= 4 {
			year = result.PublicationDate[:4]
		}
		yearInt := 0
		if year != "" {
			yearInt, _ = strconv.Atoi(year)
		}

		venue := strings.TrimSpace(result.JournalName)

		// Collect author names
		authorNames := make([]string, 0, len(result.Authors))
		for _, a := range result.Authors {
			name := strings.TrimSpace(a.Name)
			if name != "" {
				authorNames = append(authorNames, name)
			}
		}

		metadata := map[string]string{
			"product_type": result.ProductType,
		}
		if result.Subjects != "" {
			metadata["subjects"] = result.Subjects
		}
		if len(authorNames) > 0 {
			metadata["authors"] = strings.Join(authorNames, "; ")
		}

		records = append(records, SourceRecord{
			Source:   "osti",
			SourceID: idStr,
			Title:    title,
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:       yearInt,
			Abstract:   "",
			Venue:      venue,
			URLs:       nonEmptyStrings(doiURL(doi)),
			OpenAccess: true,
			Metadata:   metadata,
		})
	}
	return SourceResponse{
		Records: records,
		RawRef:  fmt.Sprintf("osti:/api/v1/records?q=%s&page=0&size=%d", url.QueryEscape(query.Terms), limit),
	}, nil
}

type ostiRecord struct {
	OstiID          int          `json:"osti_id"`
	Title           string       `json:"title"`
	DOI             string       `json:"doi"`
	PublicationDate string       `json:"publication_date"`
	JournalName     string       `json:"journal_name"`
	ProductType     string       `json:"product_type"`
	Authors         []ostiAuthor `json:"authors"`
	Subjects        string       `json:"subjects"`
}

type ostiAuthor struct {
	Name            string `json:"name"`
	AffiliationName string `json:"affiliation_name"`
	ORCID           string `json:"orcid"`
}
