package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/core"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/fields"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/transport"
	"net/url"
	"strconv"
	"strings"
)

// ZenodoConnector searches the Zenodo open repository.
type ZenodoConnector struct {
	http transport.HTTPClient
}

// NewZenodoConnector creates a Zenodo source connector.
func NewZenodoConnector(httpClient transport.HTTPClient) ZenodoConnector {
	return ZenodoConnector{http: httpClient}
}

// Name returns the connector source name.
func (ZenodoConnector) Name() string { return "zenodo" }

// Search queries the Zenodo REST API and normalizes results into SourceRecords.
func (c ZenodoConnector) Search(ctx context.Context, query core.SourceQuery) (core.SourceResponse, error) {
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
		return core.SourceResponse{}, err
	}
	var payload zenodoSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return core.SourceResponse{}, err
	}
	records := make([]core.SourceRecord, 0, len(payload.Hits.Hits))
	for _, item := range payload.Hits.Hits {
		doi := fields.NormalizeDOI(item.DOI)
		htmlURL := strings.TrimSpace(item.Links.HTML)
		urls := fields.NonEmptyStrings(htmlURL)
		year := 0
		if len(item.Metadata.PublicationDate) >= 4 {
			year, _ = strconv.Atoi(item.Metadata.PublicationDate[:4])
		}
		license := ""
		if item.Metadata.License.ID != "" {
			license = strings.TrimSpace(item.Metadata.License.ID)
		}
		records = append(records, core.SourceRecord{
			Source:   "zenodo",
			SourceID: strconv.Itoa(item.ID),
			Title:    strings.TrimSpace(item.Metadata.Title),
			Identifiers: core.Identifiers{
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
	return core.SourceResponse{Records: records, RawRef: rawZenodoRef(params)}, nil
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
