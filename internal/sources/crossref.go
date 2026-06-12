package sources

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

// CrossrefConnector searches Crossref works.
type CrossrefConnector struct {
	http HTTPClient
}

// NewCrossrefConnector creates a Crossref source connector.
func NewCrossrefConnector(httpClient HTTPClient) CrossrefConnector {
	return CrossrefConnector{http: httpClient}
}

// Name returns the connector source name.
func (CrossrefConnector) Name() string { return "crossref" }

// Search queries Crossref works and normalizes results into SourceRecords.
func (c CrossrefConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{"query": query.Terms, "rows": strconv.Itoa(limit)}
	body, err := c.http.Get(ctx, "/works", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload crossrefWorksResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Message.Items))
	for _, work := range payload.Message.Items {
		doi := normalizeSourceDOI(work.DOI)
		records = append(records, SourceRecord{
			Source:   "crossref",
			SourceID: doi,
			Title:    firstString(work.Title),
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: doi,
			},
			Year:      crossrefYear(work),
			Abstract:  stripSimpleJATS(work.Abstract),
			Venue:     firstString(work.ContainerTitle),
			Publisher: strings.TrimSpace(work.Publisher),
			URLs:      nonEmptyStrings(work.URL),
			Metadata: map[string]string{
				"type":            strings.TrimSpace(work.Type),
				"reference_count": strconv.Itoa(work.ReferenceCount),
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawCrossrefRef(params)}, nil
}

type crossrefWorksResponse struct {
	Message crossrefMessage `json:"message"`
}

type crossrefMessage struct {
	Items []crossrefWork `json:"items"`
}

type crossrefWork struct {
	DOI             string            `json:"DOI"`
	Title           []string          `json:"title"`
	Abstract        string            `json:"abstract"`
	PublishedPrint  crossrefDateParts `json:"published-print"`
	PublishedOnline crossrefDateParts `json:"published-online"`
	Issued          crossrefDateParts `json:"issued"`
	ContainerTitle  []string          `json:"container-title"`
	Publisher       string            `json:"publisher"`
	URL             string            `json:"URL"`
	Type            string            `json:"type"`
	ReferenceCount  int               `json:"reference-count"`
}

type crossrefDateParts struct {
	DateParts [][]int `json:"date-parts"`
}

func rawCrossrefRef(params map[string]string) string {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return "crossref:/works?" + values.Encode()
}

func firstString(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return compactSpace(value)
		}
	}
	return ""
}

func crossrefYear(work crossrefWork) int {
	for _, date := range []crossrefDateParts{work.PublishedPrint, work.PublishedOnline, work.Issued} {
		if len(date.DateParts) > 0 && len(date.DateParts[0]) > 0 {
			return date.DateParts[0][0]
		}
	}
	return 0
}

func stripSimpleJATS(value string) string {
	value = strings.ReplaceAll(value, "<jats:p>", "")
	value = strings.ReplaceAll(value, "</jats:p>", "")
	return compactSpace(value)
}
