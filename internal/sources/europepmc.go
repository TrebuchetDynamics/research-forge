package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// EuropePMCConnector searches Europe PMC metadata.
type EuropePMCConnector struct {
	http HTTPClient
}

// NewEuropePMCConnector creates a Europe PMC source connector.
func NewEuropePMCConnector(httpClient HTTPClient) EuropePMCConnector {
	return EuropePMCConnector{http: httpClient}
}

// Name returns the connector source name.
func (EuropePMCConnector) Name() string { return "europepmc" }

// Search queries Europe PMC and normalizes results into SourceRecords.
func (c EuropePMCConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{"query": query.Terms, "pageSize": strconv.Itoa(limit), "format": "json"}
	body, err := c.http.Get(ctx, "/webservices/rest/search", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload europePMCSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.ResultList.Result))
	for _, item := range payload.ResultList.Result {
		pmid := strings.TrimSpace(firstNonEmpty(item.PMID, item.ID))
		fullTextURLs := item.FullTextURLList.URLs()
		records = append(records, SourceRecord{
			Source:   "europepmc",
			SourceID: pmid,
			Title:    strings.TrimSpace(item.Title),
			Identifiers: Identifiers{
				DOI:   normalizeSourceDOI(item.DOI),
				PMID:  pmid,
				PMCID: item.PMCID,
			},
			Year:       parseSourceYear(item.PubYear),
			Abstract:   strings.TrimSpace(item.AbstractText),
			Venue:      strings.TrimSpace(item.JournalTitle),
			URLs:       fullTextURLs,
			License:    strings.TrimSpace(item.License),
			OpenAccess: strings.EqualFold(strings.TrimSpace(item.IsOpenAccess), "Y"),
			Metadata: map[string]string{
				"author_string": strings.TrimSpace(item.AuthorString),
				"mesh_terms":    strings.Join(item.MeshHeadingList.Terms(), "; "),
				"full_text_url": firstNonEmpty(fullTextURLs...),
				"license":       strings.TrimSpace(item.License),
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawEuropePMCRef(params)}, nil
}

type europePMCSearchResponse struct {
	ResultList struct {
		Result []europePMCResult `json:"result"`
	} `json:"resultList"`
}

type europePMCResult struct {
	ID              string                   `json:"id"`
	PMID            string                   `json:"pmid"`
	PMCID           string                   `json:"pmcid"`
	DOI             string                   `json:"doi"`
	Title           string                   `json:"title"`
	AuthorString    string                   `json:"authorString"`
	JournalTitle    string                   `json:"journalTitle"`
	PubYear         string                   `json:"pubYear"`
	AbstractText    string                   `json:"abstractText"`
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
	return nonEmptyStrings(urls...)
}

func (l europePMCMeshHeadingList) Terms() []string {
	terms := []string{}
	for _, heading := range l.Headings {
		term := strings.TrimSpace(firstNonEmpty(heading.DescriptorName, heading.Term))
		if term != "" {
			terms = append(terms, term)
		}
	}
	return terms
}

func rawEuropePMCRef(params map[string]string) string {
	values := url.Values{}
	for _, key := range []string{"format", "pageSize", "query"} {
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
