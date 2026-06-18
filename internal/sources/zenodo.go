package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ZenodoConnector searches the Zenodo open repository.
type ZenodoConnector struct {
	http HTTPClient
}

// NewZenodoConnector creates a Zenodo source connector.
func NewZenodoConnector(httpClient HTTPClient) ZenodoConnector {
	return ZenodoConnector{http: httpClient}
}

// Name returns the connector source name.
func (ZenodoConnector) Name() string { return "zenodo" }

// Search queries the Zenodo REST API and normalizes results into SourceRecords.
func (c ZenodoConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"q":    query.Terms,
		"size": strconv.Itoa(limit),
		"sort": "bestmatch",
	}
	body, err := c.http.Get(ctx, "/api/records", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload zenodoSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Hits.Hits))
	for _, item := range payload.Hits.Hits {
		doi := normalizeSourceDOI(item.DOI)
		htmlURL := strings.TrimSpace(item.Links.HTML)
		urls := nonEmptyStrings(htmlURL)
		year := 0
		if len(item.Metadata.PublicationDate) >= 4 {
			year, _ = strconv.Atoi(item.Metadata.PublicationDate[:4])
		}
		license := ""
		if item.Metadata.License.ID != "" {
			license = strings.TrimSpace(item.Metadata.License.ID)
		}
		records = append(records, SourceRecord{
			Source:   "zenodo",
			SourceID: strconv.Itoa(item.ID),
			Title:    strings.TrimSpace(item.Metadata.Title),
			Identifiers: Identifiers{
				DOI: doi,
			},
			Year:       year,
			Abstract:   strings.TrimSpace(item.Metadata.Description),
			Venue:      strings.TrimSpace(item.Metadata.ResourceType.Title),
			URLs:       urls,
			License:    license,
			OpenAccess: strings.EqualFold(strings.TrimSpace(item.Metadata.AccessRight), "open"),
			Metadata: map[string]string{
				"resource_type": strings.TrimSpace(item.Metadata.ResourceType.Type),
				"access_right":  strings.TrimSpace(item.Metadata.AccessRight),
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawZenodoRef(params)}, nil
}

type zenodoSearchResponse struct {
	Hits struct {
		Hits []zenodoRecord `json:"hits"`
	} `json:"hits"`
}

type zenodoRecord struct {
	ID       int    `json:"id"`
	DOI      string `json:"doi"`
	Metadata struct {
		Title           string `json:"title"`
		Description     string `json:"description"`
		PublicationDate string `json:"publication_date"`
		AccessRight     string `json:"access_right"`
		License         struct {
			ID string `json:"id"`
		} `json:"license"`
		ResourceType struct {
			Title string `json:"title"`
			Type  string `json:"type"`
		} `json:"resource_type"`
	} `json:"metadata"`
	Links struct {
		HTML string `json:"html"`
	} `json:"links"`
}

func rawZenodoRef(params map[string]string) string {
	values := url.Values{}
	for _, key := range []string{"q", "size", "sort"} {
		if v := strings.TrimSpace(params[key]); v != "" {
			values.Set(key, v)
		}
	}
	return fmt.Sprintf("zenodo:/api/records?%s", values.Encode())
}
