package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// InspireHEPConnector searches the INSPIRE HEP literature database.
type InspireHEPConnector struct {
	http HTTPClient
}

// NewInspireHEPConnector creates an INSPIRE HEP source connector.
func NewInspireHEPConnector(httpClient HTTPClient) InspireHEPConnector {
	return InspireHEPConnector{http: httpClient}
}

// Name returns the connector source name.
func (InspireHEPConnector) Name() string { return "inspire-hep" }

// Search queries the INSPIRE HEP literature API and normalizes results into SourceRecords.
func (c InspireHEPConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"q":    query.Terms,
		"size": strconv.Itoa(limit),
		"sort": "mostrecent",
	}
	body, err := c.http.Get(ctx, "/api/literature", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload inspireHEPSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Hits.Hits))
	for _, item := range payload.Hits.Hits {
		doi := ""
		if len(item.Metadata.DOIs) > 0 {
			doi = normalizeSourceDOI(item.Metadata.DOIs[0].Value)
		}
		arxivID := ""
		if len(item.Metadata.ArXivEprints) > 0 {
			arxivID = strings.TrimSpace(item.Metadata.ArXivEprints[0].Value)
		}
		title := ""
		if len(item.Metadata.Titles) > 0 {
			title = strings.TrimSpace(item.Metadata.Titles[0].Title)
		}
		abstract := ""
		if len(item.Metadata.Abstracts) > 0 {
			abstract = strings.TrimSpace(item.Metadata.Abstracts[0].Value)
		}
		venue := ""
		year := 0
		if len(item.Metadata.PublicationInfo) > 0 {
			venue = strings.TrimSpace(item.Metadata.PublicationInfo[0].JournalTitle)
			year = item.Metadata.PublicationInfo[0].Year
		}
		htmlURL := fmt.Sprintf("https://inspirehep.net/literature/%s", strings.TrimSpace(item.ID))
		records = append(records, SourceRecord{
			Source:   "inspire-hep",
			SourceID: strings.TrimSpace(item.ID),
			Title:    title,
			Identifiers: Identifiers{
				DOI:     doi,
				ArXivID: arxivID,
			},
			Year:       year,
			Abstract:   abstract,
			Venue:      venue,
			URLs:       nonEmptyStrings(htmlURL),
			OpenAccess: arxivID != "",
			Metadata: map[string]string{
				"document_type": inspireDocType(item.Metadata.DocumentType),
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawInspireHEPRef(params)}, nil
}

func inspireDocType(types []string) string {
	if len(types) == 0 {
		return ""
	}
	return strings.Join(types, ", ")
}

type inspireHEPSearchResponse struct {
	Hits struct {
		Total int `json:"total"`
		Hits  []inspireHEPHit `json:"hits"`
	} `json:"hits"`
}

type inspireHEPHit struct {
	ID       string `json:"id"`
	Metadata struct {
		Titles       []struct{ Title string `json:"title"` }         `json:"titles"`
		Abstracts    []struct{ Value string `json:"value"` }         `json:"abstracts"`
		DOIs         []struct{ Value string `json:"value"` }         `json:"dois"`
		ArXivEprints []struct{ Value string `json:"value"` }         `json:"arxiv_eprints"`
		PublicationInfo []struct {
			JournalTitle string `json:"journal_title"`
			Year         int    `json:"year"`
		} `json:"publication_info"`
		DocumentType []string `json:"document_type"`
	} `json:"metadata"`
}

func rawInspireHEPRef(params map[string]string) string {
	values := url.Values{}
	for _, key := range []string{"q", "size", "sort"} {
		if v := strings.TrimSpace(params[key]); v != "" {
			values.Set(key, v)
		}
	}
	return fmt.Sprintf("inspire-hep:/api/literature?%s", values.Encode())
}
