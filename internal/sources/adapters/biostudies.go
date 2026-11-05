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

// BioStudiesConnector searches the EMBL-EBI BioStudies database.
//
// BioStudies (ebi.ac.uk/biostudies) is a database of life-sciences studies
// operated by the European Bioinformatics Institute (EMBL-EBI). It aggregates
// studies from Europe PMC, ArrayExpress, and other EBI repositories. The search
// API requires no authentication.
type BioStudiesConnector struct {
	http transport.HTTPClient
}

// NewBioStudiesConnector creates a BioStudies source connector.
func NewBioStudiesConnector(httpClient transport.HTTPClient) BioStudiesConnector {
	return BioStudiesConnector{http: httpClient}
}

// Name returns the connector source name.
func (BioStudiesConnector) Name() string { return "biostudies" }

// Search queries the BioStudies search API and normalizes results into SourceRecords.
func (c BioStudiesConnector) Search(ctx context.Context, query core.SourceQuery) (core.SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"query":    query.Terms,
		"pageSize": strconv.Itoa(limit),
		"page":     "1",
	}
	body, err := c.http.Get(ctx, "/biostudies/api/v1/search", params)
	if err != nil {
		return core.SourceResponse{}, err
	}
	var payload biostudiesSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return core.SourceResponse{}, err
	}
	records := make([]core.SourceRecord, 0, len(payload.Hits))
	for _, hit := range payload.Hits {
		title := strings.TrimSpace(hit.Title)
		if title == "" {
			continue
		}
		year := 0
		if len(hit.ReleaseDate) >= 4 {
			year, _ = strconv.Atoi(hit.ReleaseDate[:4])
		}
		studyURL := ""
		if hit.Accession != "" {
			studyURL = "https://www.ebi.ac.uk/biostudies/studies/" + hit.Accession
		}
		metadata := map[string]string{}
		if hit.Author != "" {
			metadata["authors_raw"] = hit.Author
		}
		if hit.Type != "" {
			metadata["study_type"] = hit.Type
		}
		records = append(records, core.SourceRecord{
			Source:   "biostudies",
			SourceID: hit.Accession,
			Title:    title,
			Identifiers: core.Identifiers{
				CrossrefID: "biostudies:" + hit.Accession,
			},
			Year:       year,
			OpenAccess: hit.IsPublic,
			URLs:       fields.NonEmptyStrings(studyURL),
			Metadata:   metadata,
		})
	}
	rawRef := fmt.Sprintf("biostudies:/biostudies/api/v1/search?query=%s&pageSize=%d&page=1", url.QueryEscape(query.Terms), limit)
	return core.SourceResponse{Records: records, RawRef: rawRef}, nil
}

type biostudiesSearchResponse struct {
	Hits      []biostudiesHit `json:"hits"`
	TotalHits int             `json:"totalHits"`
}

type biostudiesHit struct {
	Accession   string `json:"accession"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	ReleaseDate string `json:"release_date"`
	IsPublic    bool   `json:"isPublic"`
	Content     string `json:"content"`
}
