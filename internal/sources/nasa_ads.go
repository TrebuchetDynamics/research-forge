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

func (NASAADSConnector) Name() string { return "ads" }

func (c NASAADSConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	params := map[string]string{"q": strings.TrimSpace(query.Terms), "rows": strconv.Itoa(limit), "fl": "bibcode,title,doi,year,author,pub"}
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
		metadata := map[string]string{"bibcode": bibcode}
		if pub := compactADSString(doc.Pub); pub != "" {
			metadata["publication"] = pub
		}
		records = append(records, SourceRecord{Source: "ads", SourceID: bibcode, Title: title, Identifiers: Identifiers{DOI: doi}, Year: year, Venue: compactADSString(doc.Pub), Metadata: metadata})
	}
	return SourceResponse{Records: records, RawRef: "ads:/v1/search/query?q=" + params["q"]}, nil
}

type nasaADSPayload struct {
	Response struct {
		Docs []nasaADSDoc `json:"docs"`
	} `json:"response"`
}

type nasaADSDoc struct {
	Bibcode string   `json:"bibcode"`
	Title   []string `json:"title"`
	DOI     []string `json:"doi"`
	Year    string   `json:"year"`
	Pub     string   `json:"pub"`
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
