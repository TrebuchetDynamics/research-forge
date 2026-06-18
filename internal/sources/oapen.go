package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// OAPenConnector searches the OAPEN Library for open access books.
//
// OAPEN (Open Access Publishing in European Networks, library.oapen.org) is
// the leading repository of peer-reviewed open access academic books and
// book chapters, covering humanities, social sciences, and STEM. The DSpace
// REST API requires no authentication.
type OAPenConnector struct {
	http HTTPClient
}

// NewOAPenConnector creates an OAPEN Library source connector.
func NewOAPenConnector(httpClient HTTPClient) OAPenConnector {
	return OAPenConnector{http: httpClient}
}

// Name returns the connector source name.
func (OAPenConnector) Name() string { return "oapen" }

// Search queries the OAPEN DSpace REST search endpoint and normalizes results
// into SourceRecords. Each item's metadata is a flat key-value array; this
// connector picks out the standard DC and OAPEN fields.
func (c OAPenConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"query":  query.Terms,
		"limit":  strconv.Itoa(limit),
		"offset": "0",
		"expand": "metadata",
	}
	body, err := c.http.Get(ctx, "/rest/search", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var items []oapenItem
	if err := json.Unmarshal(body, &items); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(items))
	for _, item := range items {
		// Build lookup maps from the flat metadata array.
		// metaFirst holds the first value seen for each key.
		// metaAll holds all values (for multi-valued fields like authors).
		metaFirst := map[string]string{}
		metaAll := map[string][]string{}
		for _, m := range item.Metadata {
			k := m.Key
			v := strings.TrimSpace(m.Value)
			if v == "" {
				continue
			}
			if _, exists := metaFirst[k]; !exists {
				metaFirst[k] = v
			}
			metaAll[k] = append(metaAll[k], v)
		}
		title := metaFirst["dc.title"]
		if title == "" {
			title = strings.TrimSpace(item.Name)
		}
		if title == "" {
			continue
		}
		// OAPEN stores DOIs under "oapen.identifier.doi"; values may be full
		// URL (https://doi.org/…) or bare DOI — normalizeSourceDOI handles both.
		doi := normalizeSourceDOI(metaFirst["oapen.identifier.doi"])
		crossrefID := ""
		if doi == "" {
			crossrefID = "oapen:" + item.Handle
		}
		year := 0
		if y := metaFirst["dc.date.issued"]; len(y) >= 4 {
			year, _ = strconv.Atoi(y[:4])
		}
		var authorNames []string
		for _, a := range metaAll["dc.contributor.author"] {
			if a != "" {
				authorNames = append(authorNames, a)
			}
		}
		metadata := map[string]string{}
		if len(authorNames) > 0 {
			metadata["authors"] = strings.Join(authorNames, "; ")
		}
		if lang := metaFirst["dc.language"]; lang != "" {
			metadata["language"] = lang
		}
		records = append(records, SourceRecord{
			Source:   "oapen",
			SourceID: item.Handle,
			Title:    title,
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:       year,
			Abstract:   metaFirst["dc.description.abstract"],
			Publisher:  metaFirst["publisher.name"],
			OpenAccess: true,
			Metadata:   metadata,
		})
	}
	rawRef := fmt.Sprintf("oapen:/rest/search?query=%s&limit=%d&offset=0", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type oapenItem struct {
	UUID     string      `json:"uuid"`
	Name     string      `json:"name"`
	Handle   string      `json:"handle"`
	Type     string      `json:"type"`
	Metadata []oapenMeta `json:"metadata"`
}

type oapenMeta struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
