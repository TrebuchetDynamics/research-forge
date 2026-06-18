package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// DataCiteConnector searches the DataCite research data DOI registry.
type DataCiteConnector struct {
	http HTTPClient
}

// NewDataCiteConnector creates a DataCite source connector.
func NewDataCiteConnector(httpClient HTTPClient) DataCiteConnector {
	return DataCiteConnector{http: httpClient}
}

// Name returns the connector source name.
func (DataCiteConnector) Name() string { return "datacite" }

// Search queries the DataCite REST API and normalizes results into SourceRecords.
func (c DataCiteConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"query":      query.Terms,
		"page[size]": strconv.Itoa(limit),
	}
	body, err := c.http.Get(ctx, "/dois", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload dataciteSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Data))
	for _, item := range payload.Data {
		attr := item.Attributes
		doi := normalizeSourceDOI(attr.DOI)

		// First non-empty title.
		title := ""
		for _, t := range attr.Titles {
			if s := strings.TrimSpace(t.Title); s != "" {
				title = s
				break
			}
		}

		// First Abstract description.
		abstract := ""
		for _, d := range attr.Descriptions {
			if d.DescriptionType == "Abstract" {
				if s := strings.TrimSpace(d.Description); s != "" {
					abstract = s
					break
				}
			}
		}

		// License and OA detection.
		license := ""
		openAccess := false
		for _, r := range attr.RightsList {
			if license == "" {
				if s := strings.TrimSpace(r.Rights); s != "" {
					license = s
				}
			}
			if strings.Contains(r.RightsURI, "creativecommons.org") {
				openAccess = true
			}
		}

		resourceType := strings.TrimSpace(attr.Types.ResourceTypeGeneral)

		records = append(records, SourceRecord{
			Source:   "datacite",
			SourceID: doi,
			Title:    title,
			Identifiers: Identifiers{
				DOI: doi,
			},
			Year:       attr.PublicationYear,
			Abstract:   abstract,
			Venue:      resourceType,
			Publisher:  strings.TrimSpace(attr.Publisher),
			URLs:       nonEmptyStrings(strings.TrimSpace(attr.URL)),
			License:    license,
			OpenAccess: openAccess,
			Metadata: map[string]string{
				"resource_type": resourceType,
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawDataCiteRef(query.Terms, limit)}, nil
}

type dataciteSearchResponse struct {
	Data []dataciteItem `json:"data"`
	Meta struct {
		Total int `json:"total"`
	} `json:"meta"`
}

type dataciteItem struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Attributes dataciteAttributes `json:"attributes"`
}

type dataciteAttributes struct {
	DOI             string `json:"doi"`
	Titles          []struct {
		Title string `json:"title"`
	} `json:"titles"`
	Descriptions []struct {
		Description     string `json:"description"`
		DescriptionType string `json:"descriptionType"`
	} `json:"descriptions"`
	PublicationYear int    `json:"publicationYear"`
	Publisher       string `json:"publisher"`
	Types           struct {
		ResourceTypeGeneral string `json:"resourceTypeGeneral"`
	} `json:"types"`
	RightsList []struct {
		Rights    string `json:"rights"`
		RightsURI string `json:"rightsUri"`
	} `json:"rightsList"`
	URL string `json:"url"`
}

func rawDataCiteRef(terms string, limit int) string {
	return fmt.Sprintf("datacite:/dois?query=%s&page[size]=%d", terms, limit)
}
