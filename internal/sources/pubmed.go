package sources

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// PubMedConnector searches PubMed through NCBI E-utilities JSON endpoints.
type PubMedConnector struct {
	http   HTTPClient
	apiKey string
	tool   string
	email  string
}

// PubMedOptions configures optional NCBI E-utilities identification parameters.
type PubMedOptions struct {
	APIKey string
	Tool   string
	Email  string
}

// NewPubMedConnector creates a PubMed source connector.
func NewPubMedConnector(httpClient HTTPClient) PubMedConnector {
	return NewPubMedConnectorWithOptions(httpClient, PubMedOptions{})
}

// NewPubMedConnectorWithOptions creates a PubMed source connector with optional NCBI identification.
func NewPubMedConnectorWithOptions(httpClient HTTPClient, options PubMedOptions) PubMedConnector {
	return PubMedConnector{
		http:   httpClient,
		apiKey: strings.TrimSpace(options.APIKey),
		tool:   strings.TrimSpace(options.Tool),
		email:  strings.TrimSpace(options.Email),
	}
}

// Name returns the connector source name.
func (PubMedConnector) Name() string { return "pubmed" }

// Search queries PubMed ESearch then ESummary and normalizes results into SourceRecords.
func (c PubMedConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	esearchParams := c.withNCBIParams(map[string]string{"db": "pubmed", "term": query.Terms, "retmax": strconv.Itoa(limit), "retmode": "json"})
	body, err := c.http.Get(ctx, "/entrez/eutils/esearch.fcgi", esearchParams)
	if err != nil {
		return SourceResponse{}, err
	}
	var search pubMedSearchResponse
	if err := json.Unmarshal(body, &search); err != nil {
		return SourceResponse{}, err
	}
	ids := normalizePubMedIDs(search.ESearchResult.IDList)
	if len(ids) == 0 {
		return SourceResponse{RawRef: rawPubMedRef(esearchParams)}, nil
	}
	summaryParams := c.withNCBIParams(map[string]string{"db": "pubmed", "id": strings.Join(ids, ","), "retmode": "json"})
	summaryBody, err := c.http.Get(ctx, "/entrez/eutils/esummary.fcgi", summaryParams)
	if err != nil {
		return SourceResponse{}, err
	}
	var summary pubMedSummaryResponse
	if err := json.Unmarshal(summaryBody, &summary); err != nil {
		return SourceResponse{}, err
	}
	meshByPMID, err := c.fetchMeshTerms(ctx, ids)
	if err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(summary.Result.UIDs))
	for _, uid := range summary.Result.UIDs {
		item, ok := summary.Result.Items[uid]
		if !ok {
			continue
		}
		pmid := strings.TrimSpace(firstNonEmpty(item.UID, uid))
		records = append(records, SourceRecord{
			Source:   "pubmed",
			SourceID: pmid,
			Title:    strings.TrimSpace(item.Title),
			Identifiers: Identifiers{
				DOI:   normalizeSourceDOI(pubMedArticleIDValue(item.ArticleIDs, "doi")),
				PMID:  pmid,
				PMCID: pubMedArticleIDValue(item.ArticleIDs, "pmc"),
			},
			Year:  parsePubMedYear(item.PubDate),
			Venue: strings.TrimSpace(item.FullJournalName),
			Metadata: map[string]string{
				"mesh_terms": strings.Join(meshByPMID[pmid], "; "),
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawPubMedRef(esearchParams)}, nil
}

type pubMedSearchResponse struct {
	ESearchResult struct {
		IDList []string `json:"idlist"`
	} `json:"esearchresult"`
}

type pubMedSummaryResponse struct {
	Result pubMedSummaryResult `json:"result"`
}

type pubMedSummaryResult struct {
	UIDs  []string                 `json:"uids"`
	Items map[string]pubMedSummary `json:"-"`
}

func (r *pubMedSummaryResult) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	_ = json.Unmarshal(raw["uids"], &r.UIDs)
	r.Items = map[string]pubMedSummary{}
	for key, value := range raw {
		if key == "uids" {
			continue
		}
		var item pubMedSummary
		if err := json.Unmarshal(value, &item); err == nil && strings.TrimSpace(item.Title) != "" {
			r.Items[key] = item
		}
	}
	return nil
}

type pubMedSummary struct {
	UID             string            `json:"uid"`
	Title           string            `json:"title"`
	FullJournalName string            `json:"fulljournalname"`
	PubDate         string            `json:"pubdate"`
	ArticleIDs      []pubMedArticleID `json:"articleids"`
}

type pubMedArticleID struct {
	IDType string `json:"idtype"`
	Value  string `json:"value"`
}

func (c PubMedConnector) fetchMeshTerms(ctx context.Context, ids []string) (map[string][]string, error) {
	if len(ids) == 0 {
		return map[string][]string{}, nil
	}
	params := c.withNCBIParams(map[string]string{"db": "pubmed", "id": strings.Join(ids, ","), "retmode": "xml"})
	body, err := c.http.Get(ctx, "/entrez/eutils/efetch.fcgi", params)
	if err != nil {
		return nil, err
	}
	var payload pubMedArticleSet
	if err := xml.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for _, article := range payload.Articles {
		pmid := strings.TrimSpace(article.PMID)
		if pmid == "" {
			continue
		}
		for _, heading := range article.MeshHeadings {
			term := strings.TrimSpace(heading.DescriptorName)
			if term != "" {
				out[pmid] = append(out[pmid], term)
			}
		}
	}
	return out, nil
}

type pubMedArticleSet struct {
	Articles []pubMedArticleXML `xml:"PubmedArticle"`
}

type pubMedArticleXML struct {
	PMID         string              `xml:"MedlineCitation>PMID"`
	MeshHeadings []pubMedMeshHeading `xml:"MedlineCitation>MeshHeadingList>MeshHeading"`
}

type pubMedMeshHeading struct {
	DescriptorName string `xml:"DescriptorName"`
}

func (c PubMedConnector) withNCBIParams(params map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range params {
		out[key] = value
	}
	if c.apiKey != "" {
		out["api_key"] = c.apiKey
	}
	if c.tool != "" {
		out["tool"] = c.tool
	}
	if c.email != "" {
		out["email"] = c.email
	}
	return out
}

func normalizePubMedIDs(ids []string) []string {
	out := []string{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func pubMedArticleIDValue(ids []pubMedArticleID, idType string) string {
	for _, id := range ids {
		if strings.EqualFold(strings.TrimSpace(id.IDType), idType) {
			return strings.TrimSpace(id.Value)
		}
	}
	return ""
}

func parsePubMedYear(pubDate string) int {
	fields := strings.Fields(strings.TrimSpace(pubDate))
	if len(fields) == 0 {
		return 0
	}
	year, _ := strconv.Atoi(fields[0])
	return year
}

func rawPubMedRef(params map[string]string) string {
	values := url.Values{}
	for _, key := range []string{"db", "email", "retmax", "retmode", "term", "tool"} {
		if value := strings.TrimSpace(params[key]); value != "" {
			values.Set(key, value)
		}
	}
	return fmt.Sprintf("pubmed:/entrez/eutils/esearch.fcgi?%s", values.Encode())
}
