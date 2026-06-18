package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// nberYearRe extracts a 4-digit year from NBER's displaydate field
// (e.g. "January 2006", "October 2021").
var nberYearRe = regexp.MustCompile(`\b(\d{4})\b`)

// nberPaperNumRe extracts the paper number from a URL path like /papers/w34276.
var nberPaperNumRe = regexp.MustCompile(`/papers/(w\d+)`)

// NBERConnector searches NBER Working Papers via the NBER site search API.
//
// NBER (National Bureau of Economic Research, nber.org) publishes working
// papers across all areas of economics. Each working paper carries an official
// DOI of the form 10.3386/w<number>, which this connector constructs from
// the paper URL. The site search API requires no authentication.
type NBERConnector struct {
	http HTTPClient
}

// NewNBERConnector creates an NBER Working Papers source connector.
func NewNBERConnector(httpClient HTTPClient) NBERConnector {
	return NBERConnector{http: httpClient}
}

// Name returns the connector source name.
func (NBERConnector) Name() string { return "nber" }

// Search queries the NBER working-paper search API and normalizes results into
// SourceRecords. DOIs are constructed from the paper URL path; author HTML is
// stripped; year is parsed from the human-readable date field.
func (c NBERConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	// The NBER search API has a minimum page size of 50. We request at least
	// that many and slice to the requested limit after receiving results.
	perPage := limit
	if perPage < 50 {
		perPage = 50
	}
	params := map[string]string{
		"q":       query.Terms,
		"perPage": strconv.Itoa(perPage),
		"page":    "1",
	}
	body, err := c.http.Get(ctx, "/api/v1/working_page_listing/contentType/working_paper/_/_/search", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload nberSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	results := payload.Results
	if len(results) > limit {
		results = results[:limit]
	}
	records := make([]SourceRecord, 0, len(results))
	for _, r := range results {
		title := strings.TrimSpace(r.Title)
		if title == "" {
			continue
		}
		// Construct DOI from URL path (e.g. /papers/w34276 → 10.3386/w34276).
		paperNum := ""
		if m := nberPaperNumRe.FindStringSubmatch(r.URL); m != nil {
			paperNum = m[1]
		}
		doi := ""
		if paperNum != "" {
			doi = normalizeSourceDOI("10.3386/" + paperNum)
		}
		crossrefID := ""
		if doi == "" {
			if paperNum != "" {
				crossrefID = "nber:" + paperNum
			} else {
				continue // no usable identifier
			}
		}
		year := 0
		if m := nberYearRe.FindStringSubmatch(r.DisplayDate); m != nil {
			year, _ = strconv.Atoi(m[1])
		}
		// Authors come as HTML-encoded strings (e.g. `<a href="/people/...">Name</a>`);
		// strip all tags to extract plain names.
		var authorNames []string
		for _, aHTML := range r.Authors {
			name := strings.TrimSpace(htmlTagRe.ReplaceAllString(aHTML, ""))
			if name != "" {
				authorNames = append(authorNames, name)
			}
		}
		metadata := map[string]string{}
		if len(authorNames) > 0 {
			metadata["authors"] = strings.Join(authorNames, "; ")
		}
		if paperNum != "" {
			metadata["nber_id"] = paperNum
		}
		fullURL := ""
		if r.URL != "" {
			fullURL = "https://www.nber.org" + r.URL
		}
		records = append(records, SourceRecord{
			Source:   "nber",
			SourceID: paperNum,
			Title:    title,
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:       year,
			Abstract:   strings.TrimSpace(r.Abstract),
			URLs:       nonEmptyStrings(fullURL),
			OpenAccess: false, // NBER papers require subscription (free after 18-month embargo)
			Metadata:   metadata,
		})
	}
	rawRef := fmt.Sprintf("nber:/api/v1/working_page_listing/contentType/working_paper/_/_/search?q=%s&perPage=%d&page=1", url.QueryEscape(query.Terms), perPage)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type nberSearchResponse struct {
	TotalResults int          `json:"totalResults"`
	Results      []nberResult `json:"results"`
}

type nberResult struct {
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Authors     []string `json:"authors"`
	DisplayDate string   `json:"displaydate"`
	Abstract    string   `json:"abstract"`
}
