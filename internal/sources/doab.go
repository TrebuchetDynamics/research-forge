package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DOABConnector searches the Directory of Open Access Books (DOAB).
//
// DOAB is a community-driven database of open access books. All listed books
// are freely available and peer-reviewed. The REST API returns DSpace-style
// metadata with Dublin Core and OAPEN-specific fields.
type DOABConnector struct {
	http HTTPClient
}

// NewDOABConnector creates a DOAB source connector.
func NewDOABConnector(httpClient HTTPClient) DOABConnector {
	return DOABConnector{http: httpClient}
}

// Name returns the connector source name.
func (DOABConnector) Name() string { return "doab" }

// Search queries the DOAB REST API and normalizes results into SourceRecords.
func (c DOABConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"query":  query.Terms,
		"expand": "metadata",
		"limit":  strconv.Itoa(limit),
	}
	body, err := c.http.Get(ctx, "/rest/search", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var items []doabItem
	if err := json.Unmarshal(body, &items); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(items))
	for _, item := range items {
		meta := buildDOABMetaIndex(item.Metadata)

		title := strings.TrimSpace(meta.first("dc.title"))
		if title == "" {
			continue
		}

		doi := normalizeSourceDOI(meta.first("oapen.identifier.doi"))
		ids := Identifiers{DOI: doi}
		if doi == "" {
			ids.CrossrefID = "doab:" + item.Handle
		}

		year := 0
		if dateStr := meta.first("dc.date.issued"); len(dateStr) >= 4 {
			year, _ = strconv.Atoi(dateStr[:4])
		}

		abstract := strings.TrimSpace(meta.first("dc.description.abstract"))

		publisher := strings.TrimSpace(meta.first("publisher.name"))
		if publisher == "" {
			publisher = strings.TrimSpace(meta.first("oapen.imprint"))
		}

		license := strings.TrimSpace(meta.first("publisher.oalicense"))

		itemURL := strings.TrimSpace(meta.first("dc.identifier.uri"))

		// Collect all author and editor values.
		var contributors []string
		contributors = append(contributors, meta.all("dc.contributor.author")...)
		contributors = append(contributors, meta.all("dc.contributor.editor")...)

		metadata := map[string]string{}
		if len(contributors) > 0 {
			metadata["authors"] = strings.Join(contributors, "; ")
		}

		records = append(records, SourceRecord{
			Source:      "doab",
			SourceID:    item.Handle,
			Title:       title,
			Identifiers: ids,
			Year:        year,
			Abstract:    abstract,
			Publisher:   publisher,
			License:     license,
			OpenAccess:  true,
			URLs:        nonEmptyStrings(itemURL),
			Metadata:    metadata,
		})
	}
	rawRef := fmt.Sprintf("doab:/rest/search?query=%s&limit=%d", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

// doabItem is a single DOAB REST API result.
type doabItem struct {
	UUID     string          `json:"uuid"`
	Handle   string          `json:"handle"`
	Name     string          `json:"name"`
	Type     string          `json:"type"`
	Metadata []doabMetaEntry `json:"metadata"`
}

// doabMetaEntry is one key/value metadata pair in a DOAB item.
type doabMetaEntry struct {
	Key      string  `json:"key"`
	Value    string  `json:"value"`
	Language *string `json:"language"`
}

// doabMetaIndex is a lookup helper built from a DOAB item's metadata array.
type doabMetaIndex map[string][]string

func buildDOABMetaIndex(entries []doabMetaEntry) doabMetaIndex {
	idx := make(doabMetaIndex)
	for _, e := range entries {
		idx[e.Key] = append(idx[e.Key], e.Value)
	}
	return idx
}

// first returns the first value for the given key, or "" if absent.
func (idx doabMetaIndex) first(key string) string {
	if vals := idx[key]; len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// all returns all values for the given key.
func (idx doabMetaIndex) all(key string) []string {
	return idx[key]
}
