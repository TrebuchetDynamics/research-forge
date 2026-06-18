package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// BASEConnector searches the Bielefeld Academic Search Engine (BASE).
type BASEConnector struct {
	http HTTPClient
}

// NewBASEConnector creates a BASE source connector.
func NewBASEConnector(httpClient HTTPClient) BASEConnector {
	return BASEConnector{http: httpClient}
}

// Name returns the connector source name.
func (BASEConnector) Name() string { return "base" }

// Search queries the BASE HTTP search interface and normalizes results into SourceRecords.
func (c BASEConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"func":   "PerformSearch",
		"query":  query.Terms,
		"hits":   strconv.Itoa(limit),
		"offset": "0",
		"format": "json",
	}
	body, err := c.http.Get(ctx, "/cgi-bin/BaseHttpSearchInterface.fcgi", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload baseSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Response.Docs))
	for _, doc := range payload.Response.Docs {
		if len(doc.DCTitle) == 0 {
			continue
		}
		title := strings.TrimSpace(doc.DCTitle[0])
		if title == "" {
			continue
		}

		sourceID := ""
		for _, id := range doc.DCIdentifier {
			if strings.HasPrefix(id, "oai:") {
				sourceID = id
				break
			}
		}
		if sourceID == "" && len(doc.DCIdentifier) > 0 {
			sourceID = doc.DCIdentifier[0]
		}

		doi := ""
		for _, id := range doc.DCIdentifier {
			if strings.HasPrefix(id, "doi:") {
				doi = normalizeSourceDOI(strings.TrimPrefix(id, "doi:"))
				break
			}
		}
		crossrefID := ""
		if doi == "" {
			crossrefID = sourceID
		}

		year := 0
		if len(doc.DCDate) > 0 && len(doc.DCDate[0]) >= 4 {
			year, _ = strconv.Atoi(doc.DCDate[0][:4])
		}

		abstract := ""
		if len(doc.DCDescription) > 0 {
			abstract = strings.TrimSpace(doc.DCDescription[0])
		}

		venue := ""
		if len(doc.DCSource) > 0 {
			venue = strings.TrimSpace(doc.DCSource[0])
		}

		publisher := ""
		if len(doc.DCPublisher) > 0 {
			publisher = strings.TrimSpace(doc.DCPublisher[0])
		}

		license := ""
		openAccess := false
		for _, r := range doc.DCRights {
			if license == "" {
				license = strings.TrimSpace(r)
			}
			if strings.Contains(strings.ToLower(r), "cc") {
				openAccess = true
			}
		}

		dcType := ""
		if len(doc.DCType) > 0 {
			dcType = strings.TrimSpace(doc.DCType[0])
		}

		records = append(records, SourceRecord{
			Source:   "base",
			SourceID: sourceID,
			Title:    title,
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:      year,
			Abstract:  abstract,
			Venue:     venue,
			Publisher: publisher,
			URLs:      nonEmptyStrings(doc.DCLink),
			License:   license,
			OpenAccess: openAccess,
			Metadata: map[string]string{
				"type":         dcType,
				"contributors": strings.Join(doc.DCContributor, "; "),
			},
		})
	}
	return SourceResponse{
		Records: records,
		RawRef:  fmt.Sprintf("base:/cgi-bin/BaseHttpSearchInterface.fcgi?func=PerformSearch&query=%s&hits=%d", query.Terms, limit),
	}, nil
}

type baseSearchResponse struct {
	Response struct {
		NumFound int           `json:"numFound"`
		Start    int           `json:"start"`
		Docs     []baseDoc     `json:"docs"`
	} `json:"response"`
}

type baseDoc struct {
	DCTitle       []string `json:"dctitle"`
	DCDescription []string `json:"dcdescription"`
	DCIdentifier  []string `json:"dcidentifier"`
	DCDate        []string `json:"dcdate"`
	DCContributor []string `json:"dccontributor"`
	DCSource      []string `json:"dcsource"`
	DCPublisher   []string `json:"dcpublisher"`
	DCRights      []string `json:"dcrights"`
	DCFormat      []string `json:"dcformat"`
	DCType        []string `json:"dctype"`
	DCLink        string   `json:"dclink"`
}
