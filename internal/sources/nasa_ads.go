package sources

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
)

// NASAADSConnector searches NASA ADS Solr records for physics/astronomy bibliography normalization.
type NASAADSConnector struct{ http HTTPClient }

func NewNASAADSConnector(httpClient HTTPClient) NASAADSConnector {
	return NASAADSConnector{http: httpClient}
}

func NewNASAADSHTTPClient(baseURL, token string) HTTPClient {
	headers := map[string]string{}
	if strings.TrimSpace(token) != "" {
		headers["Authorization"] = "Bearer " + strings.TrimSpace(token)
	}
	return NewHTTPClient(HTTPClientOptions{BaseURL: baseURL, Headers: headers})
}

func RedactNASAADSToken(value string) string {
	if strings.Contains(value, "Bearer ") {
		return value[:strings.Index(value, "Bearer ")+len("Bearer ")] + "[REDACTED_ADS_TOKEN]"
	}
	return value
}

func (NASAADSConnector) Name() string { return "ads" }

func (c NASAADSConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	params := map[string]string{"q": strings.TrimSpace(query.Terms), "rows": strconv.Itoa(limit), "fl": "bibcode,title,doi,year,author,pub,abstract,doctype,database"}
	data, err := c.http.Get(ctx, "/v1/search/query", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload nasaADSPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := []SourceRecord{}
	for _, doc := range payload.Response.Docs {
		title := compactADSString(firstADSValue(doc.Title))
		doi := normalizeSourceDOI(firstADSValue(doc.DOI))
		bibcode := strings.TrimSpace(doc.Bibcode)
		if title == "" && doi == "" && bibcode == "" {
			continue
		}
		year, _ := strconv.Atoi(strings.TrimSpace(doc.Year))
		metadata := map[string]string{"bibcode": bibcode, "doctype": strings.TrimSpace(doc.DocType), "database": strings.Join(nonEmptyStrings(doc.Database...), ",")}
		if pub := compactADSString(doc.Pub); pub != "" {
			metadata["publication"] = pub
		}
		records = append(records, SourceRecord{Source: "ads", SourceID: bibcode, Title: title, Identifiers: Identifiers{DOI: doi, ADSBibcode: bibcode}, Year: year, Abstract: strings.TrimSpace(doc.Abstract), Venue: compactADSString(doc.Pub), Metadata: metadata})
	}
	return SourceResponse{Records: records, RawRef: "ads:/v1/search/query?q=" + params["q"]}, nil
}

func (c NASAADSConnector) ExpandCitationGraph(ctx context.Context, bibcode string, limit int) (CitationGraphExpansion, error) {
	bibcode = strings.TrimSpace(bibcode)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	params := map[string]string{"q": "bibcode:" + bibcode, "rows": "1", "fl": "bibcode,title,reference,citation"}
	data, err := c.http.Get(ctx, "/v1/search/query", params)
	if err != nil {
		return CitationGraphExpansion{}, err
	}
	var payload nasaADSPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return CitationGraphExpansion{}, err
	}
	expansion := CitationGraphExpansion{SeedID: bibcode, Records: map[string]SourceRecord{}, RawRef: "ads:/v1/search/query?q=" + params["q"]}
	if len(payload.Response.Docs) == 0 {
		return expansion, nil
	}
	doc := payload.Response.Docs[0]
	expansion.Records[bibcode] = SourceRecord{Source: "ads", SourceID: bibcode, Title: compactADSString(firstADSValue(doc.Title)), Identifiers: Identifiers{ADSBibcode: bibcode}, Metadata: map[string]string{"bibcode": bibcode}}
	for _, ref := range firstNStrings(doc.Reference, limit) {
		expansion.Edges = append(expansion.Edges, CitationEdge{SourceID: bibcode, TargetID: ref})
	}
	for _, citation := range firstNStrings(doc.Citation, limit) {
		expansion.Edges = append(expansion.Edges, CitationEdge{SourceID: citation, TargetID: bibcode})
	}
	return expansion, nil
}

func firstNStrings(values []string, limit int) []string {
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
		if len(out) >= limit {
			break
		}
	}
	return out
}

type nasaADSPayload struct {
	Response struct {
		Docs []nasaADSDoc `json:"docs"`
	} `json:"response"`
}

type nasaADSDoc struct {
	Bibcode   string   `json:"bibcode"`
	Title     []string `json:"title"`
	DOI       []string `json:"doi"`
	Year      string   `json:"year"`
	Pub       string   `json:"pub"`
	Abstract  string   `json:"abstract"`
	DocType   string   `json:"doctype"`
	Database  []string `json:"database"`
	Reference []string `json:"reference"`
	Citation  []string `json:"citation"`
}

func firstADSValue(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func compactADSString(value string) string { return strings.Join(strings.Fields(value), " ") }
