package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// htmlTagRe matches HTML tags for stripping.
var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// stripHTMLTags removes HTML tags from s and collapses whitespace.
func stripHTMLTags(s string) string {
	result := htmlTagRe.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(result), " ")
}

// DryadConnector searches the Dryad research data repository.
//
// Dryad (datadryad.org) is an open-access repository for scientific research
// data. It provides a HAL-style REST API with no authentication required.
type DryadConnector struct {
	http HTTPClient
}

// NewDryadConnector creates a Dryad source connector.
func NewDryadConnector(httpClient HTTPClient) DryadConnector {
	return DryadConnector{http: httpClient}
}

// Name returns the connector source name.
func (DryadConnector) Name() string { return "dryad" }

// Search queries the Dryad REST API and normalizes results into SourceRecords.
func (c DryadConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"q":        query.Terms,
		"per_page": strconv.Itoa(limit),
	}
	body, err := c.http.Get(ctx, "/api/v2/search", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload dryadSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Embedded.Datasets))
	for _, ds := range payload.Embedded.Datasets {
		title := strings.TrimSpace(ds.Title)
		if title == "" {
			continue
		}

		// Extract DOI by stripping "doi:" prefix from the identifier field.
		doi := normalizeSourceDOI(strings.TrimPrefix(ds.Identifier, "doi:"))

		year := 0
		if len(ds.PublicationDate) >= 4 {
			year, _ = strconv.Atoi(ds.PublicationDate[:4])
		}

		abstract := stripHTMLTags(ds.Abstract)

		// Build authors list as "LastName, FirstName".
		var authorParts []string
		for _, a := range ds.Authors {
			name := strings.TrimSpace(a.LastName + ", " + a.FirstName)
			name = strings.TrimSuffix(strings.TrimPrefix(name, ", "), ", ")
			if name != "" && name != "," {
				authorParts = append(authorParts, name)
			}
		}

		metadata := map[string]string{}
		if len(authorParts) > 0 {
			metadata["authors"] = strings.Join(authorParts, "; ")
		}
		if ds.FieldOfScience != "" {
			metadata["field_of_science"] = ds.FieldOfScience
		}

		records = append(records, SourceRecord{
			Source:   "dryad",
			SourceID: ds.Identifier,
			Title:    title,
			Identifiers: Identifiers{
				DOI: doi,
			},
			Year:       year,
			Abstract:   abstract,
			Venue:      "",
			Publisher:  "Dryad",
			License:    ds.License,
			OpenAccess: true,
			URLs:       nonEmptyStrings(ds.SharingLink),
			Metadata:   metadata,
		})
	}
	rawRef := fmt.Sprintf("dryad:/api/v2/search?q=%s&per_page=%d", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

// dryadSearchResponse is the top-level HAL-style response from Dryad.
type dryadSearchResponse struct {
	Count    int `json:"count"`
	Total    int `json:"total"`
	Embedded struct {
		Datasets []dryadDataset `json:"stash:datasets"`
	} `json:"_embedded"`
}

// dryadDataset is a single Dryad dataset entry.
type dryadDataset struct {
	Identifier      string        `json:"identifier"`
	ID              int           `json:"id"`
	Title           string        `json:"title"`
	Authors         []dryadAuthor `json:"authors"`
	Abstract        string        `json:"abstract"`
	FieldOfScience  string        `json:"fieldOfScience"`
	PublicationDate string        `json:"publicationDate"`
	SharingLink     string        `json:"sharingLink"`
	License         string        `json:"license"`
}

// dryadAuthor is a single author in a Dryad dataset.
type dryadAuthor struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	ORCID       string `json:"orcid"`
	Affiliation string `json:"affiliation"`
}
