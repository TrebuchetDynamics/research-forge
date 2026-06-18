package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// PMCConnector searches PubMed Central (PMC) via the NCBI E-utilities API.
//
// PMC (www.ncbi.nlm.nih.gov/pmc/) is NCBI's free full-text archive of
// biomedical and life sciences journal articles. The connector uses a two-step
// flow: esearch retrieves PMC IDs for a query, then esummary fetches metadata
// for those IDs. Both endpoints are part of the public NCBI E-utilities API
// and require no authentication for moderate-volume use.
type PMCConnector struct {
	http HTTPClient
}

// NewPMCConnector creates a PubMed Central source connector.
func NewPMCConnector(httpClient HTTPClient) PMCConnector {
	return PMCConnector{http: httpClient}
}

// Name returns the connector source name.
func (PMCConnector) Name() string { return "pmc" }

// Search queries the NCBI E-utilities esearch endpoint for PMC, then fetches
// metadata via esummary, and normalizes results into SourceRecords.
func (c PMCConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	rawRef := fmt.Sprintf("pmc:/entrez/eutils/esearch.fcgi?db=pmc&term=%s&retmax=%d&sort=relevance", url.QueryEscape(query.Terms), limit)

	// Step 1: esearch — get a ranked list of PMC IDs.
	searchParams := map[string]string{
		"db":      "pmc",
		"term":    query.Terms,
		"retmax":  strconv.Itoa(limit),
		"retmode": "json",
		"sort":    "relevance",
	}
	searchBody, err := c.http.Get(ctx, "/entrez/eutils/esearch.fcgi", searchParams)
	if err != nil {
		return SourceResponse{}, err
	}
	var searchResult pmcSearchResult
	if err := json.Unmarshal(searchBody, &searchResult); err != nil {
		return SourceResponse{}, err
	}
	ids := searchResult.ESearchResult.IDList
	if len(ids) == 0 {
		return SourceResponse{RawRef: rawRef}, nil
	}

	// Step 2: esummary — fetch metadata for each PMC ID.
	summaryParams := map[string]string{
		"db":      "pmc",
		"id":      strings.Join(ids, ","),
		"retmode": "json",
	}
	summaryBody, err := c.http.Get(ctx, "/entrez/eutils/esummary.fcgi", summaryParams)
	if err != nil {
		return SourceResponse{}, err
	}
	// The result map mixes PMC IDs (object values) with the "uids" key (array
	// value). Use RawMessage so we can skip non-summary entries gracefully.
	var rawSummary struct {
		Result map[string]json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(summaryBody, &rawSummary); err != nil {
		return SourceResponse{}, err
	}

	records := make([]SourceRecord, 0, len(ids))
	for _, id := range ids {
		raw, ok := rawSummary.Result[id]
		if !ok {
			continue
		}
		var sum pmcSummary
		if err := json.Unmarshal(raw, &sum); err != nil {
			continue
		}
		title := strings.TrimSpace(sum.Title)
		if title == "" {
			continue
		}
		doi := ""
		pmcID := ""
		for _, aid := range sum.ArticleIDs {
			switch aid.IDType {
			case "doi":
				doi = normalizeSourceDOI(aid.Value)
			case "pmcid":
				pmcID = aid.Value
			}
		}
		crossrefID := ""
		if doi == "" {
			if pmcID != "" {
				crossrefID = "pmc:" + pmcID
			} else {
				crossrefID = "pmc:" + id
			}
		}
		year := 0
		if len(sum.PubDate) >= 4 {
			year, _ = strconv.Atoi(sum.PubDate[:4])
		}
		var authorNames []string
		for _, a := range sum.Authors {
			if a.Name != "" && a.AuthType == "Author" {
				authorNames = append(authorNames, a.Name)
			}
		}
		metadata := map[string]string{}
		if len(authorNames) > 0 {
			metadata["authors"] = strings.Join(authorNames, "; ")
		}
		if pmcID != "" {
			metadata["pmc_id"] = pmcID
		}
		if sum.Volume != "" {
			metadata["volume"] = sum.Volume
		}
		if sum.Issue != "" {
			metadata["issue"] = sum.Issue
		}
		if sum.Pages != "" {
			metadata["pages"] = sum.Pages
		}
		records = append(records, SourceRecord{
			Source:   "pmc",
			SourceID: id,
			Title:    title,
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:       year,
			Venue:      sum.Source,
			OpenAccess: true,
			Metadata:   metadata,
		})
	}
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type pmcSearchResult struct {
	ESearchResult struct {
		Count  string   `json:"count"`
		IDList []string `json:"idlist"`
	} `json:"esearchresult"`
}

type pmcSummary struct {
	Title      string          `json:"title"`
	PubDate    string          `json:"pubdate"`
	Source     string          `json:"source"`
	Authors    []pmcAuthor     `json:"authors"`
	Volume     string          `json:"volume"`
	Issue      string          `json:"issue"`
	Pages      string          `json:"pages"`
	ArticleIDs []pmcArticleID  `json:"articleids"`
}

type pmcAuthor struct {
	Name     string `json:"name"`
	AuthType string `json:"authtype"`
}

type pmcArticleID struct {
	IDType string `json:"idtype"`
	Value  string `json:"value"`
}
