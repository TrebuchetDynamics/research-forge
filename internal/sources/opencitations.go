package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// OpenCitationsConnector searches the OpenCitations COCI citation graph.
type OpenCitationsConnector struct {
	http HTTPClient
}

// NewOpenCitationsConnector creates an OpenCitations source connector.
func NewOpenCitationsConnector(httpClient HTTPClient) OpenCitationsConnector {
	return OpenCitationsConnector{http: httpClient}
}

// Name returns the connector source name.
func (OpenCitationsConnector) Name() string { return "opencitations" }

// Search queries the OpenCitations COCI API for papers that cite the DOI in
// query.Terms, then fetches metadata for the citing papers.
func (c OpenCitationsConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}

	doi := normalizeSourceDOI(query.Terms)
	citationsPath := fmt.Sprintf("/index/api/v2/citations/%s", doi)
	rawRef := rawOpenCitationsRef(doi)

	body, err := c.http.Get(ctx, citationsPath, nil)
	if err != nil {
		return SourceResponse{}, err
	}

	var citations []openCitationsCitationRecord
	if err := json.Unmarshal(body, &citations); err != nil {
		return SourceResponse{}, err
	}

	if len(citations) == 0 {
		return SourceResponse{Records: []SourceRecord{}, RawRef: rawRef}, nil
	}

	// Collect citing DOIs (strip "doi:" prefix), up to limit.
	citingDOIs := make([]string, 0, limit)
	creationByDOI := map[string]string{}
	timespanByDOI := map[string]string{}
	for _, c := range citations {
		if len(citingDOIs) >= limit {
			break
		}
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

	// Batch-fetch metadata for citing DOIs.
	metadataPath := fmt.Sprintf("/index/api/v2/metadata/%s", strings.Join(citingDOIs, ";"))
	metaBody, err := c.http.Get(ctx, metadataPath, nil)
	if err != nil {
		return SourceResponse{}, err
	}

	var metaItems []openCitationsMetadataRecord
	if err := json.Unmarshal(metaBody, &metaItems); err != nil {
		return SourceResponse{}, err
	}

	records := make([]SourceRecord, 0, len(metaItems))
	for _, item := range metaItems {
		itemDOI := normalizeSourceDOI(item.ID)
		year := 0
		if len(item.PubDate) >= 4 {
			year, _ = strconv.Atoi(item.PubDate[:4])
		}
		records = append(records, SourceRecord{
			Source:   "opencitations",
			SourceID: itemDOI,
			Title:    strings.TrimSpace(item.Title),
			Identifiers: Identifiers{
				DOI: itemDOI,
			},
			Year:     year,
			Abstract: "",
			Venue:    strings.TrimSpace(item.Venue),
			Publisher: strings.TrimSpace(item.Publisher),
			URLs:     nonEmptyStrings(doiURL(itemDOI)),
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

type openCitationsCitationRecord struct {
	OCI      string `json:"oci"`
	Citing   string `json:"citing"`
	Cited    string `json:"cited"`
	Creation string `json:"creation"`
	Timespan string `json:"timespan"`
	JournalSC string `json:"journal_sc"`
	AuthorSC  string `json:"author_sc"`
}

type openCitationsMetadataRecord struct {
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

func rawOpenCitationsRef(doi string) string {
	return fmt.Sprintf("opencitations:/index/api/v2/citations/%s", doi)
}

func doiURL(doi string) string {
	if doi == "" {
		return ""
	}
	return "https://doi.org/" + doi
}
