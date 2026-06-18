package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// OSFConnector searches OSF Preprints, which aggregates PsyArXiv, SocArXiv,
// EarthArXiv, and other community preprint servers under a single API.
type OSFConnector struct {
	http HTTPClient
}

// NewOSFConnector creates an OSF source connector.
func NewOSFConnector(httpClient HTTPClient) OSFConnector {
	return OSFConnector{http: httpClient}
}

// Name returns the connector source name.
func (OSFConnector) Name() string { return "osf" }

// Search queries the OSF Preprints API and normalizes results into SourceRecords.
func (c OSFConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"filter[title]": query.Terms,
		"page[size]":    strconv.Itoa(limit),
	}
	body, err := c.http.Get(ctx, "/v2/preprints/", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload osfResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Data))
	for _, item := range payload.Data {
		attrs := item.Attributes
		doi := normalizeSourceDOI(strings.TrimSpace(attrs.PreprintDOI))
		htmlURL := strings.TrimSpace(item.Links.HTML)
		year := 0
		if len(attrs.DatePublished) >= 4 {
			year, _ = strconv.Atoi(attrs.DatePublished[:4])
		}
		license := ""
		if attrs.License.Name != "" {
			license = strings.TrimSpace(attrs.License.Name)
		}
		osfID := strings.TrimSpace(item.ID)
		// OSF preprints without a DOI use their OSF ID via CrossrefID so that
		// library.PaperRecords passes identifier validation.
		crossrefID := ""
		if doi == "" {
			crossrefID = osfID
		}
		records = append(records, SourceRecord{
			Source:   "osf",
			SourceID: osfID,
			Title:    strings.TrimSpace(attrs.Title),
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:       year,
			Abstract:   strings.TrimSpace(attrs.Description),
			Venue:      "OSF Preprints",
			URLs:       nonEmptyStrings(htmlURL),
			License:    license,
			OpenAccess: true,
			Metadata: map[string]string{
				"tags": strings.Join(attrs.Tags, "; "),
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawOSFRef(params)}, nil
}

type osfResponse struct {
	Data  []osfPreprint `json:"data"`
	Links struct {
		Next string `json:"next"`
	} `json:"links"`
}

type osfPreprint struct {
	ID         string `json:"id"`
	Attributes struct {
		Title         string   `json:"title"`
		Description   string   `json:"description"`
		DatePublished string   `json:"date_published"`
		Tags          []string `json:"tags"`
		PreprintDOI   string   `json:"preprint_doi"`
		License       struct {
			Name string `json:"name"`
		} `json:"license"`
	} `json:"attributes"`
	Links struct {
		HTML string `json:"html"`
	} `json:"links"`
}

func rawOSFRef(params map[string]string) string {
	values := url.Values{}
	for _, key := range []string{"filter[title]", "page[size]"} {
		if v := strings.TrimSpace(params[key]); v != "" {
			values.Set(key, v)
		}
	}
	return fmt.Sprintf("osf:/v2/preprints/?%s", values.Encode())
}
