package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ERICConnector searches the Education Resources Information Center (ERIC).
//
// ERIC is the US Department of Education's bibliographic database covering
// education research and practice (2M+ records). It has no DOI field in the
// search API; SourceID (EJ/ED number) is stored in CrossrefID as "eric:<id>".
type ERICConnector struct {
	http HTTPClient
}

// NewERICConnector creates an ERIC source connector.
func NewERICConnector(httpClient HTTPClient) ERICConnector {
	return ERICConnector{http: httpClient}
}

// Name returns the connector source name.
func (ERICConnector) Name() string { return "eric" }

// Search queries the ERIC API and normalizes results into SourceRecords.
func (c ERICConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"search": query.Terms,
		"rows":   strconv.Itoa(limit),
		"format": "json",
		"fields": "id,title,author,description,subject,publicationdateyear,url,peerreviewed,publicationtype",
	}
	body, err := c.http.Get(ctx, "/", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload ericSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Response.Docs))
	for _, doc := range payload.Response.Docs {
		title := strings.TrimSpace(doc.Title)
		if title == "" {
			continue
		}
		docURL := strings.TrimSpace(doc.URL)
		if docURL == "" && doc.ID != "" {
			docURL = "https://eric.ed.gov/?id=" + doc.ID
		}
		var subjects []string
		for _, s := range doc.Subject {
			if s = strings.TrimSpace(s); s != "" {
				subjects = append(subjects, s)
			}
		}
		var pubTypes []string
		for _, pt := range doc.PublicationType {
			if pt = strings.TrimSpace(pt); pt != "" {
				pubTypes = append(pubTypes, pt)
			}
		}
		records = append(records, SourceRecord{
			Source:      "eric",
			SourceID:    doc.ID,
			Title:       title,
			Identifiers: Identifiers{CrossrefID: "eric:" + doc.ID},
			Year:        doc.PublicationDateYear,
			Abstract:    strings.TrimSpace(doc.Description),
			OpenAccess:  docURL != "",
			URLs:        nonEmptyStrings(docURL),
			Metadata: map[string]string{
				"subjects":        strings.Join(subjects, "; "),
				"peerreviewed":    doc.PeerReviewed,
				"publicationtype": strings.Join(pubTypes, "; "),
			},
		})
	}
	rawRef := fmt.Sprintf("eric:/?search=%s&rows=%d", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type ericSearchResponse struct {
	Response struct {
		NumFound int       `json:"numFound"`
		Docs     []ericDoc `json:"docs"`
	} `json:"response"`
}

type ericDoc struct {
	ID                  string   `json:"id"`
	Title               string   `json:"title"`
	Author              []string `json:"author"`
	Description         string   `json:"description"`
	Subject             []string `json:"subject"`
	PublicationDateYear int      `json:"publicationdateyear"`
	URL                 string   `json:"url"`
	PeerReviewed        string   `json:"peerreviewed"`
	PublicationType     []string `json:"publicationtype"`
}
