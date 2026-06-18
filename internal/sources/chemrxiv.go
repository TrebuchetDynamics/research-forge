package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ChemRxivConnector searches ChemRxiv, the chemistry preprint server.
//
// ChemRxiv is hosted by the American Chemical Society and powered by the
// Cambridge Open Engage platform. It covers chemistry, chemical biology, and
// related fields. All records are openly accessible preprints; license is
// populated from the Cambridge Open Engage license field when present.
type ChemRxivConnector struct {
	http HTTPClient
}

// NewChemRxivConnector creates a ChemRxiv source connector.
func NewChemRxivConnector(httpClient HTTPClient) ChemRxivConnector {
	return ChemRxivConnector{http: httpClient}
}

// Name returns the connector source name.
func (ChemRxivConnector) Name() string { return "chemrxiv" }

// Search queries the ChemRxiv Cambridge Open Engage API and normalizes results.
func (c ChemRxivConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"term":  query.Terms,
		"limit": strconv.Itoa(limit),
		"skip":  "0",
	}
	body, err := c.http.Get(ctx, "/engage/chemrxiv/public-api/v1/items", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload chemrxivSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.ItemHits))
	for _, hit := range payload.ItemHits {
		item := hit.Item
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		doi := normalizeSourceDOI(item.DOI)
		ids := Identifiers{DOI: doi}
		if doi == "" && item.ID != "" {
			ids.CrossrefID = "chemrxiv:" + item.ID
		}
		year := 0
		if len(item.StatusDate) >= 4 {
			year, _ = strconv.Atoi(item.StatusDate[:4])
		}
		category := ""
		if len(item.Categories) > 0 {
			category = strings.TrimSpace(item.Categories[0].Name)
		}
		license := strings.TrimSpace(item.License.Name)
		records = append(records, SourceRecord{
			Source:      "chemrxiv",
			SourceID:    item.ID,
			Title:       title,
			Identifiers: ids,
			Year:        year,
			Abstract:    strings.TrimSpace(item.Abstract),
			OpenAccess:  true,
			License:     license,
			URLs:        nonEmptyStrings(doiURL(doi)),
			Metadata: map[string]string{
				"category":    category,
				"license_id":  item.License.ID,
				"license_url": item.License.URL,
			},
		})
	}
	rawRef := fmt.Sprintf("chemrxiv:/engage/chemrxiv/public-api/v1/items?term=%s&limit=%d", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type chemrxivSearchResponse struct {
	ItemHits   []chemrxivHit `json:"itemHits"`
	TotalCount int           `json:"totalCount"`
}

type chemrxivHit struct {
	Item chemrxivItem `json:"item"`
}

type chemrxivItem struct {
	ID         string `json:"id"`
	DOI        string `json:"doi"`
	Title      string `json:"title"`
	Abstract   string `json:"abstract"`
	StatusDate string `json:"statusDate"`
	Authors    []struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
	} `json:"authors"`
	Categories []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"categories"`
	License struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"license"`
}
