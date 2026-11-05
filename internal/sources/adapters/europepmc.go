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

// EuropePMCConnector searches Europe PMC metadata.
type EuropePMCConnector struct {
	http transport.HTTPClient
}

// NewEuropePMCConnector creates a Europe PMC source connector.
func NewEuropePMCConnector(httpClient transport.HTTPClient) EuropePMCConnector {
	return EuropePMCConnector{http: httpClient}
}

// Name returns the connector source name.
func (EuropePMCConnector) Name() string { return "europepmc" }

// Search queries Europe PMC and normalizes results into SourceRecords.
func (c EuropePMCConnector) Search(ctx context.Context, query core.SourceQuery) (core.SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	terms := query.Terms
	if filter := strings.TrimSpace(query.Filters["filter"]); filter != "" {
		terms = "(" + terms + ") AND " + filter
	}
	cursor := query.PageCursor
	if cursor == "" {
		cursor = "*"
	}
	params := map[string]string{"query": terms, "pageSize": strconv.Itoa(limit), "format": "json", "resultType": "core", "cursorMark": cursor}
	body, err := c.http.Get(ctx, "/webservices/rest/search", params)
	if err != nil {
		return core.SourceResponse{}, err
	}
	var payload europePMCSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return core.SourceResponse{}, err
	}
	records := make([]core.SourceRecord, 0, len(payload.ResultList.Result))
	for _, item := range payload.ResultList.Result {
		sourceID := strings.TrimSpace(fields.FirstNonEmpty(item.ID, item.PMID))
		// Only MED's native ID is a PubMed identifier. Older responses may
		// omit source but provide an explicit pmid field.
		pmid := strings.TrimSpace(item.PMID)
		if strings.EqualFold(strings.TrimSpace(item.Source), "MED") {
			pmid = sourceID
		} else if strings.TrimSpace(item.Source) != "" {
			pmid = ""
		}
		if strings.IndexFunc(pmid, func(r rune) bool { return r < '0' || r > '9' }) >= 0 {
			pmid = ""
		}
		fullTextURLs := item.FullTextURLList.URLs()
		records = append(records, core.SourceRecord{
			Source:   "europepmc",
			SourceID: sourceID,
			Title:    strings.TrimSpace(item.Title),
			Identifiers: core.Identifiers{
				DOI:   fields.NormalizeDOI(item.DOI),
				PMID:  pmid,
				PMCID: item.PMCID,
			},
			Year:       parseSourceYear(item.PubYear),
			Abstract:   strings.TrimSpace(item.AbstractText),
			Venue:      strings.TrimSpace(fields.FirstNonEmpty(item.JournalInfo.Journal.Title, item.JournalTitle)),
			URLs:       fullTextURLs,
			License:    strings.TrimSpace(item.License),
			OpenAccess: strings.EqualFold(strings.TrimSpace(item.IsOpenAccess), "Y"),
			Metadata: map[string]string{
				"author_string":     strings.TrimSpace(item.AuthorString),
				"mesh_terms":        strings.Join(item.MeshHeadingList.Terms(), "; "),
				"full_text_url":     fields.FirstNonEmpty(fullTextURLs...),
				"license":           strings.TrimSpace(item.License),
				"publication_dates": strings.Join(fields.NonEmptyStrings(item.FirstPublicationDate, item.ElectronicPublicationDate, item.JournalInfo.PrintPublicationDate), ";"),
			},
		})
	}
	return core.SourceResponse{Records: records, RawRef: rawEuropePMCRef(params), NextPageCursor: strings.TrimSpace(payload.NextCursorMark)}, nil
}

type europePMCSearchResponse struct {
	NextCursorMark string `json:"nextCursorMark"`
	ResultList     struct {
		Result []europePMCResult `json:"result"`
	} `json:"resultList"`
}

type europePMCResult struct {
	Source                    string `json:"source"`
	ID                        string `json:"id"`
	PMID                      string `json:"pmid"`
	PMCID                     string `json:"pmcid"`
	DOI                       string `json:"doi"`
	Title                     string `json:"title"`
	AuthorString              string `json:"authorString"`
	JournalTitle              string `json:"journalTitle"`
	PubYear                   string `json:"pubYear"`
	AbstractText              string `json:"abstractText"`
	FirstPublicationDate      string `json:"firstPublicationDate"`
	ElectronicPublicationDate string `json:"electronicPublicationDate"`
	JournalInfo               struct {
		PrintPublicationDate string `json:"printPublicationDate"`
		Journal              struct {
			Title string `json:"title"`
		} `json:"journal"`
	} `json:"journalInfo"`
	IsOpenAccess    string                   `json:"isOpenAccess"`
	License         string                   `json:"license"`
	MeshHeadingList europePMCMeshHeadingList `json:"meshHeadingList"`
	FullTextURLList europePMCFullTextURLList `json:"fullTextUrlList"`
}

type europePMCMeshHeadingList struct {
	Headings []europePMCMeshHeading `json:"meshHeading"`
}

type europePMCFullTextURLList struct {
	URLsList []europePMCFullTextURL `json:"fullTextUrl"`
}

type europePMCFullTextURL struct {
	URL string `json:"url"`
}

type europePMCMeshHeading struct {
	DescriptorName string `json:"descriptorName"`
	Term           string `json:"term"`
}

func (l europePMCFullTextURLList) URLs() []string {
	urls := []string{}
	for _, item := range l.URLsList {
		if strings.TrimSpace(item.URL) != "" {
			urls = append(urls, item.URL)
		}
	}
	return fields.NonEmptyStrings(urls...)
}

func (l europePMCMeshHeadingList) Terms() []string {
	terms := []string{}
	for _, heading := range l.Headings {
		term := strings.TrimSpace(fields.FirstNonEmpty(heading.DescriptorName, heading.Term))
		if term != "" {
			terms = append(terms, term)
		}
	}
	return terms
}

func rawEuropePMCRef(params map[string]string) string {
	values := url.Values{}
	for _, key := range []string{"format", "pageSize", "query", "resultType", "cursorMark"} {
		if value := strings.TrimSpace(params[key]); value != "" {
			values.Set(key, value)
		}
	}
	return fmt.Sprintf("europepmc:/webservices/rest/search?%s", values.Encode())
}

func parseSourceYear(value string) int {
	year, _ := strconv.Atoi(strings.TrimSpace(value))
	return year
}
