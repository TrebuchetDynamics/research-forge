package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// OpenCitationsConnector queries the OpenCitations COCI citation graph.
//
// Query terms must be a DOI. The connector returns papers that cite that DOI
// by calling the Index v1 citations endpoint, then batch-fetches bibliographic
// metadata from the Meta v1 endpoint.
//
// API migration note (2026): the legacy /index/coci/api/v1/ paths and
// the /index/v1/metadata endpoint have been retired. The current paths are:
//   - https://api.opencitations.net/index/v1/citations/{doi}
//   - https://api.opencitations.net/meta/v1/metadata/doi:{doi1}__doi:{doi2}
type OpenCitationsConnector struct {
	http HTTPClient
}

// NewOpenCitationsConnector creates an OpenCitations source connector.
func NewOpenCitationsConnector(httpClient HTTPClient) OpenCitationsConnector {
	return OpenCitationsConnector{http: httpClient}
}

// Name returns the connector source name.
func (OpenCitationsConnector) Name() string { return "opencitations" }

// Search queries OpenCitations for papers citing the DOI in query.Terms.
func (c OpenCitationsConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	doi := normalizeSourceDOI(query.Terms)
	rawRef := fmt.Sprintf("opencitations:/index/v1/citations/%s", doi)

	body, err := c.http.Get(ctx, fmt.Sprintf("/index/v1/citations/%s", doi), nil)
	if err != nil {
		return SourceResponse{}, err
	}
	var citations []ocCitationRecord
	if err := json.Unmarshal(body, &citations); err != nil {
		return SourceResponse{}, err
	}
	if len(citations) == 0 {
		return SourceResponse{Records: []SourceRecord{}, RawRef: rawRef}, nil
	}

	citingDOIs := make([]string, 0, limit)
	creationByDOI := map[string]string{}
	timespanByDOI := map[string]string{}
	for _, c := range citations {
		if len(citingDOIs) >= limit {
			break
		}
		// citing field is a plain DOI (no prefix) in the current API
		citingDOI := normalizeSourceDOI(c.Citing)
		if citingDOI == "" {
			continue
		}
		citingDOIs = append(citingDOIs, citingDOI)
		creationByDOI[citingDOI] = c.Creation
		timespanByDOI[citingDOI] = c.Timespan
	}
	if len(citingDOIs) == 0 {
		return SourceResponse{Records: []SourceRecord{}, RawRef: rawRef}, nil
	}

	// Batch-fetch metadata: /meta/v1/metadata/doi:{doi1}__doi:{doi2}
	var prefixed []string
	for _, d := range citingDOIs {
		prefixed = append(prefixed, "doi:"+d)
	}
	metaPath := fmt.Sprintf("/meta/v1/metadata/%s", strings.Join(prefixed, "__"))
	metaBody, err := c.http.Get(ctx, metaPath, nil)
	if err != nil {
		// Metadata fetch is best-effort; return citing DOIs without titles
		records := make([]SourceRecord, 0, len(citingDOIs))
		for _, d := range citingDOIs {
			records = append(records, SourceRecord{
				Source:      "opencitations",
				SourceID:    d,
				Identifiers: Identifiers{DOI: d},
				URLs:        nonEmptyStrings(doiURL(d)),
				Metadata:    map[string]string{"creation": creationByDOI[d], "timespan": timespanByDOI[d]},
			})
		}
		return SourceResponse{Records: records, RawRef: rawRef}, nil
	}
	var metaItems []ocMetadataRecord
	if err := json.Unmarshal(metaBody, &metaItems); err != nil {
		return SourceResponse{}, err
	}

	records := make([]SourceRecord, 0, len(metaItems))
	for _, item := range metaItems {
		// id has format "doi:10.xxx omid:br/..." — extract first space-delimited token
		itemDOI := normalizeSourceDOI(strings.Fields(item.ID)[0])
		year := 0
		if len(item.PubDate) >= 4 {
			year, _ = strconv.Atoi(item.PubDate[:4])
		}
		// Venue field has format "Journal Name [issn:... omid:...]" — strip bracketed annotation
		venue := item.Venue
		if idx := strings.Index(venue, " ["); idx > 0 {
			venue = strings.TrimSpace(venue[:idx])
		}
		publisher := item.Publisher
		if idx := strings.Index(publisher, " ["); idx > 0 {
			publisher = strings.TrimSpace(publisher[:idx])
		}
		records = append(records, SourceRecord{
			Source:    "opencitations",
			SourceID:  itemDOI,
			Title:     strings.TrimSpace(item.Title),
			Identifiers: Identifiers{DOI: itemDOI},
			Year:      year,
			Venue:     venue,
			Publisher: publisher,
			URLs:      nonEmptyStrings(doiURL(itemDOI)),
			Metadata: map[string]string{
				"type":      strings.TrimSpace(item.Type),
				"author":    strings.TrimSpace(item.Author),
				"creation":  creationByDOI[itemDOI],
				"timespan":  timespanByDOI[itemDOI],
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type ocCitationRecord struct {
	OCI       string `json:"oci"`
	Citing    string `json:"citing"`
	Cited     string `json:"cited"`
	Creation  string `json:"creation"`
	Timespan  string `json:"timespan"`
	JournalSC string `json:"journal_sc"`
	AuthorSC  string `json:"author_sc"`
}

type ocMetadataRecord struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	PubDate   string `json:"pub_date"`
	Venue     string `json:"venue"`
	Volume    string `json:"volume"`
	Issue     string `json:"issue"`
	Page      string `json:"page"`
	Type      string `json:"type"`
	Publisher string `json:"publisher"`
	Editor    string `json:"editor"`
}

func doiURL(doi string) string {
	if doi == "" {
		return ""
	}
	return "https://doi.org/" + doi
}
