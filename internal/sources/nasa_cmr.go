package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// NASACMRConnector searches the NASA Common Metadata Repository (CMR).
//
// NASA CMR (cmr.earthdata.nasa.gov) is the authoritative catalog of NASA Earth
// observation and scientific data collections. It indexes datasets from NASA
// DAACs (Distributed Active Archive Centers) and partner organizations. The
// public JSON search API requires no authentication.
type NASACMRConnector struct {
	http HTTPClient
}

// NewNASACMRConnector creates a NASA CMR source connector.
func NewNASACMRConnector(httpClient HTTPClient) NASACMRConnector {
	return NASACMRConnector{http: httpClient}
}

// Name returns the connector source name.
func (NASACMRConnector) Name() string { return "nasa-cmr" }

// Search queries the NASA CMR collections endpoint and normalizes results into
// SourceRecords.
func (c NASACMRConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"keyword":   query.Terms,
		"page_size": strconv.Itoa(limit),
	}
	body, err := c.http.Get(ctx, "/search/collections.json", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload nasaCMRResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Feed.Entry))
	for _, entry := range payload.Feed.Entry {
		title := strings.TrimSpace(entry.Title)
		if title == "" {
			continue
		}
		// Extract DOI from links: look for rel ending in "/metadata#" and
		// href containing "doi.org".
		doi := ""
		for _, link := range entry.Links {
			if strings.HasSuffix(link.Rel, "/metadata#") && strings.Contains(link.Href, "doi.org") {
				doi = normalizeSourceDOI(link.Href)
				break
			}
		}
		crossrefID := ""
		if doi == "" {
			crossrefID = "nasa-cmr:" + entry.ID
		}
		year := 0
		if len(entry.Updated) >= 4 {
			year, _ = strconv.Atoi(entry.Updated[:4])
		}
		metadata := map[string]string{}
		if entry.ArchiveCenter != "" {
			metadata["archive_center"] = entry.ArchiveCenter
		}
		if len(entry.Organizations) > 0 {
			metadata["organizations"] = strings.Join(entry.Organizations, "; ")
		}
		if entry.ShortName != "" {
			metadata["short_name"] = entry.ShortName
		}
		if entry.VersionID != "" {
			metadata["version"] = entry.VersionID
		}
		records = append(records, SourceRecord{
			Source:   "nasa-cmr",
			SourceID: entry.ID,
			Title:    title,
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:       year,
			Abstract:   strings.TrimSpace(entry.Summary),
			Publisher:  "NASA",
			OpenAccess: entry.OnlineAccess,
			Metadata:   metadata,
		})
	}
	rawRef := fmt.Sprintf("nasa-cmr:/search/collections.json?keyword=%s&page_size=%d", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type nasaCMRResponse struct {
	Feed struct {
		Entry []nasaCMREntry `json:"entry"`
	} `json:"feed"`
}

type nasaCMREntry struct {
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Summary       string        `json:"summary"`
	Updated       string        `json:"updated"`
	ArchiveCenter string        `json:"archive_center"`
	Organizations []string      `json:"organizations"`
	ShortName     string        `json:"short_name"`
	VersionID     string        `json:"version_id"`
	OnlineAccess  bool          `json:"online_access_flag"`
	Links         []nasaCMRLink `json:"links"`
}

type nasaCMRLink struct {
	Rel      string `json:"rel"`
	Href     string `json:"href"`
	Hreflang string `json:"hreflang"`
}
