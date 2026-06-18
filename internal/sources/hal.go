package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// HALConnector searches the French HAL open-access archive.
//
// HAL (Hyper Articles en Ligne) is France's national OA repository with strong
// coverage in humanities, social sciences, and STEM. The API is SOLR-based and
// returns structured JSON. HAL IDs (hal-XXXXXXX) are stored in CrossrefID when
// no DOI is available.
type HALConnector struct {
	http HTTPClient
}

// NewHALConnector creates a HAL source connector.
func NewHALConnector(httpClient HTTPClient) HALConnector {
	return HALConnector{http: httpClient}
}

// Name returns the connector source name.
func (HALConnector) Name() string { return "hal" }

// Search queries the HAL open search API and normalizes results into SourceRecords.
func (c HALConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"q":    query.Terms,
		"rows": strconv.Itoa(limit),
		"fl":   "docid,halId_s,title_s,authFullName_s,producedDateY_i,abstract_s,doiId_s,journalTitle_s,publisher_s,openAccess_bool,uri_s",
		"wt":   "json",
	}
	body, err := c.http.Get(ctx, "/search/", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload halSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Response.Docs))
	for _, doc := range payload.Response.Docs {
		title := ""
		if len(doc.TitleS) > 0 {
			title = strings.TrimSpace(doc.TitleS[0])
		}
		if title == "" {
			continue
		}
		abstract := ""
		if len(doc.AbstractS) > 0 {
			abstract = strings.TrimSpace(doc.AbstractS[0])
		}
		doi := normalizeSourceDOI(doc.DOI)
		ids := Identifiers{DOI: doi}
		if doi == "" && doc.HalID != "" {
			ids.CrossrefID = "hal:" + doc.HalID
		}
		records = append(records, SourceRecord{
			Source:      "hal",
			SourceID:    doc.HalID,
			Title:       title,
			Identifiers: ids,
			Year:        doc.ProducedDateY,
			Abstract:    abstract,
			Venue:       strings.TrimSpace(doc.JournalTitle),
			Publisher:   strings.TrimSpace(doc.Publisher),
			OpenAccess:  doc.OpenAccess,
			URLs:        nonEmptyStrings(strings.TrimSpace(doc.URI)),
			Metadata:    map[string]string{"docid": doc.DocID},
		})
	}
	rawRef := fmt.Sprintf("hal:/search/?q=%s&rows=%d", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type halSearchResponse struct {
	Response struct {
		NumFound int      `json:"numFound"`
		Docs     []halDoc `json:"docs"`
	} `json:"response"`
}

type halDoc struct {
	DocID        string   `json:"docid"`
	HalID        string   `json:"halId_s"`
	TitleS       []string `json:"title_s"`
	AuthFullName []string `json:"authFullName_s"`
	ProducedDateY int     `json:"producedDateY_i"`
	AbstractS    []string `json:"abstract_s"`
	DOI          string   `json:"doiId_s"`
	JournalTitle string   `json:"journalTitle_s"`
	Publisher    string   `json:"publisher_s"`
	OpenAccess   bool     `json:"openAccess_bool"`
	URI          string   `json:"uri_s"`
}
