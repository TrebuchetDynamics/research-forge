package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// PLOSConnector searches the Public Library of Science (PLOS) article API.
type PLOSConnector struct {
	http HTTPClient
}

// NewPLOSConnector creates a PLOS source connector.
func NewPLOSConnector(httpClient HTTPClient) PLOSConnector {
	return PLOSConnector{http: httpClient}
}

// Name returns the connector source name.
func (PLOSConnector) Name() string { return "plos" }

// Search queries the PLOS search API and normalizes results into SourceRecords.
func (c PLOSConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"q":    query.Terms,
		"rows": strconv.Itoa(limit),
		"fl":   "id,title,abstract,publication_date,journal,author,article_type",
	}
	body, err := c.http.Get(ctx, "/search", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload plosSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Response.Docs))
	for _, doc := range payload.Response.Docs {
		title := strings.TrimSpace(doc.Title)
		if title == "" {
			continue
		}
		doi := normalizeSourceDOI(doc.ID)
		year := 0
		if len(doc.PublicationDate) >= 4 {
			year, _ = strconv.Atoi(doc.PublicationDate[:4])
		}
		abstract := ""
		if len(doc.Abstract) > 0 {
			abstract = doc.Abstract[0]
		}
		records = append(records, SourceRecord{
			Source:   "plos",
			SourceID: doc.ID,
			Title:    title,
			Identifiers: Identifiers{
				DOI: doi,
			},
			Year:       year,
			Abstract:   abstract,
			Venue:      strings.TrimSpace(doc.Journal),
			URLs:       nonEmptyStrings(doiURL(doi)),
			OpenAccess: true,
			Metadata: map[string]string{
				"article_type": strings.TrimSpace(doc.ArticleType),
			},
		})
	}
	rawRef := fmt.Sprintf("plos:/search?q=%s&rows=%d", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type plosSearchResponse struct {
	Response struct {
		NumFound int       `json:"numFound"`
		Docs     []plosDoc `json:"docs"`
	} `json:"response"`
}

type plosDoc struct {
	ID              string   `json:"id"`
	Title           string   `json:"title"`
	Abstract        []string `json:"abstract"`
	PublicationDate string   `json:"publication_date"`
	Journal         string   `json:"journal"`
	Author          []string `json:"author"`
	ArticleType     string   `json:"article_type"`
}
