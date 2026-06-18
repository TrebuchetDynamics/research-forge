package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ELifeConnector searches the eLife open-access journal via its content API.
//
// eLife (elifesciences.org) is a selective open-access journal covering the
// life sciences and biomedicine. All articles are fully open access. The
// public content API requires no authentication.
type ELifeConnector struct {
	http HTTPClient
}

// NewELifeConnector creates an eLife source connector.
func NewELifeConnector(httpClient HTTPClient) ELifeConnector {
	return ELifeConnector{http: httpClient}
}

// Name returns the connector source name.
func (ELifeConnector) Name() string { return "elife" }

// Search queries the eLife content search API and normalizes results into
// SourceRecords. Article titles may contain inline HTML (e.g. <sub>, <em>);
// these are stripped before storing.
func (c ELifeConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"for":      query.Terms,
		"per-page": strconv.Itoa(limit),
	}
	body, err := c.http.Get(ctx, "/search", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload elifeSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Items))
	for _, item := range payload.Items {
		// Strip any inline HTML markup from the title field.
		title := compactSpace(htmlTagRe.ReplaceAllString(item.Title, " "))
		if title == "" {
			continue
		}
		doi := normalizeSourceDOI(item.DOI)
		if doi == "" {
			continue // eLife articles without a DOI are not usable
		}
		year := 0
		if len(item.Published) >= 4 {
			year, _ = strconv.Atoi(item.Published[:4])
		}
		authorLine := strings.TrimSpace(item.AuthorLine)
		metadata := map[string]string{
			"type": strings.TrimSpace(item.Type),
		}
		if authorLine != "" {
			metadata["author_line"] = authorLine
		}
		if item.Volume != "" {
			metadata["volume"] = item.Volume
		}
		if item.ElocationID != "" {
			metadata["elocation_id"] = item.ElocationID
		}
		records = append(records, SourceRecord{
			Source:   "elife",
			SourceID: strings.TrimSpace(item.ID),
			Title:    title,
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: doi,
			},
			Year:       year,
			OpenAccess: true, // eLife is a fully open access journal
			Metadata:   metadata,
		})
	}
	rawRef := fmt.Sprintf("elife:/search?for=%s&per-page=%d", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type elifeSearchResponse struct {
	Total int           `json:"total"`
	Items []elifeItem   `json:"items"`
}

type elifeItem struct {
	ID          string `json:"id"`
	DOI         string `json:"doi"`
	Title       string `json:"title"`
	Published   string `json:"published"`
	Type        string `json:"type"`
	AuthorLine  string `json:"authorLine"`
	Volume      string `json:"volume"`
	ElocationID string `json:"elocationId"`
}
