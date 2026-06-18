package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// CrossrefConnector searches Crossref works.
type CrossrefConnector struct {
	http HTTPClient
}

// NewCrossrefConnector creates a Crossref source connector.
func NewCrossrefConnector(httpClient HTTPClient) CrossrefConnector {
	return CrossrefConnector{http: httpClient}
}

// Name returns the connector source name.
func (CrossrefConnector) Name() string { return "crossref" }

// LookupDOI fetches one Crossref work by DOI and normalizes its metadata.
func (c CrossrefConnector) LookupDOI(ctx context.Context, doi string) (SourceRecord, string, error) {
	doi = normalizeSourceDOI(doi)
	if doi == "" {
		return SourceRecord{}, "", fmt.Errorf("crossref DOI is required")
	}
	path := "/works/" + doi
	body, err := c.http.Get(ctx, path, map[string]string{})
	if err != nil {
		return SourceRecord{}, "", err
	}
	var payload crossrefWorkResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceRecord{}, "", err
	}
	return sourceRecordFromCrossrefWork(payload.Message), "crossref:" + path, nil
}

// References fetches one Crossref work and normalizes its reference-list DOI/title entries.
func (c CrossrefConnector) References(ctx context.Context, doi string) (SourceResponse, error) {
	doi = normalizeSourceDOI(doi)
	if doi == "" {
		return SourceResponse{}, fmt.Errorf("crossref DOI is required")
	}
	path := "/works/" + doi
	body, err := c.http.Get(ctx, path, map[string]string{})
	if err != nil {
		return SourceResponse{}, err
	}
	var payload crossrefWorkResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := []SourceRecord{}
	for _, ref := range payload.Message.Reference {
		refDOI := normalizeSourceDOI(ref.DOI)
		title := compactSpace(ref.ArticleTitle)
		if refDOI == "" && title == "" {
			continue
		}
		records = append(records, SourceRecord{Source: "crossref", SourceID: refDOI, Title: title, Identifiers: Identifiers{DOI: refDOI, CrossrefID: refDOI}, Metadata: map[string]string{"reference_key": strings.TrimSpace(ref.Key), "referenced_by_doi": doi}})
	}
	return SourceResponse{Records: records, RawRef: "crossref:" + path + "/references"}, nil
}

// Search queries Crossref works and normalizes results into SourceRecords.
func (c CrossrefConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{"query": query.Terms, "rows": strconv.Itoa(limit)}
	if filter := translateCrossrefFilter(query.Filters["filter"]); filter != "" {
		params["filter"] = filter
	}
	body, err := c.http.Get(ctx, "/works", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload crossrefWorksResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Message.Items))
	for _, work := range payload.Message.Items {
		records = append(records, sourceRecordFromCrossrefWork(work))
	}
	return SourceResponse{Records: records, RawRef: rawCrossrefRef(params)}, nil
}

type crossrefWorksResponse struct {
	Message crossrefMessage `json:"message"`
}

type crossrefWorkResponse struct {
	Message crossrefWork `json:"message"`
}

type crossrefMessage struct {
	Items []crossrefWork `json:"items"`
}

type crossrefWork struct {
	DOI             string                        `json:"DOI"`
	Title           []string                      `json:"title"`
	Abstract        string                        `json:"abstract"`
	PublishedPrint  crossrefDateParts             `json:"published-print"`
	PublishedOnline crossrefDateParts             `json:"published-online"`
	Issued          crossrefDateParts             `json:"issued"`
	ContainerTitle  []string                      `json:"container-title"`
	Publisher       string                        `json:"publisher"`
	URL             string                        `json:"URL"`
	Type            string                        `json:"type"`
	ReferenceCount  int                           `json:"reference-count"`
	Reference       []crossrefReference           `json:"reference"`
	Funder          []crossrefFunder              `json:"funder"`
	License         []crossrefLicense             `json:"license"`
	Relation        map[string][]crossrefRelation `json:"relation"`
}

type crossrefReference struct {
	DOI          string `json:"DOI"`
	ArticleTitle string `json:"article-title"`
	Key          string `json:"key"`
}

type crossrefFunder struct {
	Name  string   `json:"name"`
	DOI   string   `json:"DOI"`
	Award []string `json:"award"`
}

type crossrefLicense struct {
	URL string `json:"URL"`
}

type crossrefRelation struct {
	ID         string `json:"id"`
	IDType     string `json:"id-type"`
	AssertedBy string `json:"asserted-by"`
}

type crossrefDateParts struct {
	DateParts [][]int `json:"date-parts"`
}

func sourceRecordFromCrossrefWork(work crossrefWork) SourceRecord {
	doi := normalizeSourceDOI(work.DOI)
	return SourceRecord{
		Source:   "crossref",
		SourceID: doi,
		Title:    firstString(work.Title),
		Identifiers: Identifiers{
			DOI:        doi,
			CrossrefID: doi,
		},
		Year:       crossrefYear(work),
		Abstract:   stripSimpleJATS(work.Abstract),
		Venue:      firstString(work.ContainerTitle),
		Publisher:  strings.TrimSpace(work.Publisher),
		URLs:       nonEmptyStrings(work.URL),
		License:    firstCrossrefLicenseURL(work.License),
		OpenAccess: len(work.License) > 0,
		Metadata: map[string]string{
			"type":            strings.TrimSpace(work.Type),
			"reference_count": strconv.Itoa(work.ReferenceCount),
			"reference_dois":  strings.Join(crossrefReferenceDOIs(work.Reference), "; "),
			"funders":         strings.Join(crossrefFunders(work.Funder), "; "),
			"funder_awards":   strings.Join(crossrefFunderAwards(work.Funder), "; "),
			"relations":       strings.Join(crossrefRelations(work.Relation), "; "),
		},
	}
}

func firstCrossrefLicenseURL(licenses []crossrefLicense) string {
	for _, license := range licenses {
		if strings.TrimSpace(license.URL) != "" {
			return strings.TrimSpace(license.URL)
		}
	}
	return ""
}

func crossrefReferenceDOIs(references []crossrefReference) []string {
	out := []string{}
	for _, ref := range references {
		if doi := normalizeSourceDOI(ref.DOI); doi != "" {
			out = append(out, doi)
		}
	}
	return out
}

func crossrefFunders(funders []crossrefFunder) []string {
	out := []string{}
	for _, funder := range funders {
		name := compactSpace(funder.Name)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

func crossrefFunderAwards(funders []crossrefFunder) []string {
	out := []string{}
	for _, funder := range funders {
		name := compactSpace(funder.Name)
		for _, award := range funder.Award {
			award = strings.TrimSpace(award)
			if award == "" {
				continue
			}
			if name != "" {
				out = append(out, name+":"+award)
			} else {
				out = append(out, award)
			}
		}
	}
	return out
}

func crossrefRelations(relations map[string][]crossrefRelation) []string {
	out := []string{}
	for relationType, items := range relations {
		relationType = strings.TrimSpace(relationType)
		for _, item := range items {
			id := strings.TrimSpace(item.ID)
			if relationType != "" && id != "" {
				out = append(out, relationType+":"+id)
			}
		}
	}
	return out
}

func rawCrossrefRef(params map[string]string) string {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return "crossref:/works?" + values.Encode()
}

func firstString(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return compactSpace(value)
		}
	}
	return ""
}

func crossrefYear(work crossrefWork) int {
	for _, date := range []crossrefDateParts{work.PublishedPrint, work.PublishedOnline, work.Issued} {
		if len(date.DateParts) > 0 && len(date.DateParts[0]) > 0 {
			return date.DateParts[0][0]
		}
	}
	return 0
}

// translateCrossrefFilter converts a comma-separated filter string that may
// contain OpenAlex-format tokens into Crossref-native filter syntax. Tokens
// with no Crossref equivalent are dropped rather than forwarded, which prevents
// HTTP 400 responses when CLI flags like --from-year or --preset are used with
// --source crossref.
func translateCrossrefFilter(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var out []string
	for _, token := range strings.Split(raw, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		// OpenAlex: from_publication_date:YYYY-MM-DD → Crossref: from-pub-date:YYYY
		if after, ok := strings.CutPrefix(token, "from_publication_date:"); ok {
			if year := dateYear(after); year != "" {
				out = append(out, "from-pub-date:"+year)
			}
			continue
		}
		// OpenAlex: to_publication_date:YYYY-MM-DD → Crossref: until-pub-date:YYYY
		if after, ok := strings.CutPrefix(token, "to_publication_date:"); ok {
			if year := dateYear(after); year != "" {
				out = append(out, "until-pub-date:"+year)
			}
			continue
		}
		// OpenAlex type:article → Crossref type:journal-article
		if token == "type:article" {
			out = append(out, "type:journal-article")
			continue
		}
		// Drop OpenAlex-only filters that have no Crossref equivalent.
		if strings.HasPrefix(token, "is_oa:") ||
			strings.HasPrefix(token, "open_access.") ||
			strings.HasPrefix(token, "concepts.id:") {
			continue
		}
		// Pass native Crossref filter tokens through unchanged.
		out = append(out, token)
	}
	return strings.Join(out, ",")
}

// dateYear extracts the four-digit year from YYYY-MM-DD or plain YYYY.
func dateYear(s string) string {
	if len(s) >= 4 {
		year := s[:4]
		if _, err := strconv.Atoi(year); err == nil {
			return year
		}
	}
	return ""
}

func stripSimpleJATS(value string) string {
	value = strings.ReplaceAll(value, "<jats:p>", "")
	value = strings.ReplaceAll(value, "</jats:p>", "")
	return compactSpace(value)
}
