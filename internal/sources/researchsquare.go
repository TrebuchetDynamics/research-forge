package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ResearchSquareConnector searches the Research Square preprint repository.
//
// Research Square (researchsquare.com) is a preprint server serving multiple
// journal communities. It provides a public REST search API with no
// authentication required.
type ResearchSquareConnector struct {
	http HTTPClient
}

// NewResearchSquareConnector creates a Research Square source connector.
func NewResearchSquareConnector(httpClient HTTPClient) ResearchSquareConnector {
	return ResearchSquareConnector{http: httpClient}
}

// Name returns the connector source name.
func (ResearchSquareConnector) Name() string { return "researchsquare" }

// Search queries the Research Square search API and normalizes results into SourceRecords.
func (c ResearchSquareConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"term":  query.Terms,
		"limit": strconv.Itoa(limit),
		"page":  "1",
	}
	body, err := c.http.Get(ctx, "/api/search", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload rsSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Result.Data))
	for _, a := range payload.Result.Data {
		title := strings.TrimSpace(a.Title)
		if title == "" {
			continue
		}
		dv := a.DOIVersion
		if dv <= 0 {
			dv = 1
		}
		doi := ""
		crossrefID := ""
		if a.ArticleIdentity != "" {
			doi = normalizeSourceDOI(fmt.Sprintf("10.21203/rs.3.%s/v%d", a.ArticleIdentity, dv))
		}
		if doi == "" {
			crossrefID = "researchsquare:" + a.ArticleIdentity
		}
		year := 0
		if len(a.PostedAt) >= 4 {
			year, _ = strconv.Atoi(a.PostedAt[:4])
		}
		articleURL := ""
		if a.URL != "" {
			articleURL = "https://www.researchsquare.com" + a.URL
		}
		metadata := map[string]string{}
		if a.Authors != "" {
			metadata["authors_raw"] = a.Authors
		}
		if a.JournalTitle != "" {
			metadata["journal_title"] = a.JournalTitle
		}
		if a.ArticleType != "" {
			metadata["article_type"] = a.ArticleType
		}
		if a.Status != "" {
			metadata["status"] = a.Status
		}
		records = append(records, SourceRecord{
			Source:   "researchsquare",
			SourceID: a.ArticleIdentity,
			Title:    title,
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:       year,
			Publisher:  "Research Square",
			OpenAccess: true,
			URLs:       nonEmptyStrings(articleURL),
			Metadata:   metadata,
		})
	}
	rawRef := fmt.Sprintf("researchsquare:/api/search?term=%s&limit=%d&page=1", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type rsSearchResponse struct {
	Result rsResult `json:"result"`
}

type rsResult struct {
	Data  []rsArticle `json:"data"`
	Total int         `json:"total"`
}

type rsArticle struct {
	ArticleIdentity string `json:"article_identity"`
	Authors         string `json:"authors"`
	PostedAt        string `json:"posted_at"`
	DOIVersion      int    `json:"doi_version"`
	JournalTitle    string `json:"journal_title"`
	Status          string `json:"status"`
	Title           string `json:"title"`
	ArticleType     string `json:"article_type"`
	SubjectAreas    string `json:"subject_areas"`
	URL             string `json:"url"`
}
