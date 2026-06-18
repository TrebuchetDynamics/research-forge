package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DataverseConnector searches the Harvard Dataverse repository.
//
// Harvard Dataverse (dataverse.harvard.edu) is one of the largest open
// research data repositories, hosting datasets across all disciplines.
// The Dataverse REST API requires no authentication for public datasets.
type DataverseConnector struct {
	http HTTPClient
}

// NewDataverseConnector creates a Harvard Dataverse source connector.
func NewDataverseConnector(httpClient HTTPClient) DataverseConnector {
	return DataverseConnector{http: httpClient}
}

// Name returns the connector source name.
func (DataverseConnector) Name() string { return "dataverse" }

// Search queries the Harvard Dataverse search API and normalizes results into
// SourceRecords.
func (c DataverseConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"q":        query.Terms,
		"type":     "dataset",
		"per_page": strconv.Itoa(limit),
		"start":    "0",
	}
	body, err := c.http.Get(ctx, "/api/search", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload dataverseSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Data.Items))
	for _, item := range payload.Data.Items {
		title := strings.TrimSpace(item.Name)
		if title == "" {
			continue
		}
		// global_id may be "doi:10.7910/DVN/XXXXX" or "hdl:…" — only extract
		// a DOI when the prefix is explicitly "doi:".
		doi := ""
		if strings.HasPrefix(strings.ToLower(item.GlobalID), "doi:") {
			doi = normalizeSourceDOI(item.GlobalID)
		}
		crossrefID := ""
		if doi == "" {
			crossrefID = "dataverse:" + item.GlobalID
		}
		year := 0
		if len(item.PublishedAt) >= 4 {
			year, _ = strconv.Atoi(item.PublishedAt[:4])
		}
		metadata := map[string]string{}
		if len(item.Authors) > 0 {
			metadata["authors"] = strings.Join(item.Authors, "; ")
		}
		if len(item.Subjects) > 0 {
			metadata["subjects"] = strings.Join(item.Subjects, "; ")
		}
		if item.Publisher != "" {
			metadata["dataverse"] = item.Publisher
		}
		records = append(records, SourceRecord{
			Source:   "dataverse",
			SourceID: item.GlobalID,
			Title:    title,
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:       year,
			Abstract:   strings.TrimSpace(item.Description),
			Publisher:  "Harvard Dataverse",
			OpenAccess: true,
			URLs:       nonEmptyStrings(item.URL),
			Metadata:   metadata,
		})
	}
	rawRef := fmt.Sprintf("dataverse:/api/search?q=%s&type=dataset&per_page=%d&start=0", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type dataverseSearchResponse struct {
	Status string `json:"status"`
	Data   struct {
		Q          string          `json:"q"`
		TotalCount int             `json:"total_count"`
		Items      []dataverseItem `json:"items"`
	} `json:"data"`
}

type dataverseItem struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	URL         string   `json:"url"`
	GlobalID    string   `json:"global_id"`
	Description string   `json:"description"`
	PublishedAt string   `json:"published_at"`
	Publisher   string   `json:"publisher"`
	Subjects    []string `json:"subjects"`
	Authors     []string `json:"authors"`
}
