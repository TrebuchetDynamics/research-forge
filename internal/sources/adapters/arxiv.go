package adapters

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/core"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/fields"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/transport"
	"net/url"
	"strconv"
	"strings"
)

// ArXivConnector searches arXiv Atom feeds.
type ArXivConnector struct {
	http transport.HTTPClient
}

// NewArXivConnector creates an arXiv source connector.
func NewArXivConnector(httpClient transport.HTTPClient) ArXivConnector {
	return ArXivConnector{http: httpClient}
}

// Name returns the connector source name.
func (ArXivConnector) Name() string { return "arxiv" }

// LookupPaperByArXivID resolves one arXiv paper by its arXiv ID.
func (c ArXivConnector) LookupPaperByArXivID(ctx context.Context, id string) (core.SourceRecord, string, error) {
	id = fields.NormalizeArXivID(id)
	if id == "" {
		return core.SourceRecord{}, "", fmt.Errorf("arxiv id is required")
	}
	params := map[string]string{"id_list": id, "max_results": "1"}
	body, err := c.http.Get(ctx, "/api/query", params)
	if err != nil {
		return core.SourceRecord{}, "", err
	}
	var feed arxivFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return core.SourceRecord{}, "", err
	}
	if len(feed.Entries) == 0 {
		return core.SourceRecord{}, "", fmt.Errorf("arxiv: no paper found for id %s", id)
	}
	entry := feed.Entries[0]
	normID := fields.NormalizeArXivID(entry.ID)
	return core.SourceRecord{
		Source:   "arxiv",
		SourceID: normID,
		Title:    fields.CompactSpace(entry.Title),
		Abstract: fields.CompactSpace(entry.Summary),
		Year:     yearFromPublished(entry.Published),
		URLs:     arxivLinks(entry),
		Identifiers: core.Identifiers{
			ArXivID: normID,
			DOI:     fields.NormalizeDOI(entry.DOI),
		},
		Metadata: arxivMetadata(entry),
	}, rawArXivRef(params), nil
}

// Search queries arXiv and normalizes entries into SourceRecords.
func (c ArXivConnector) Search(ctx context.Context, query core.SourceQuery) (core.SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{"search_query": arxivSearchQuery(query), "max_results": strconv.Itoa(limit)}
	body, err := c.http.Get(ctx, "/api/query", params)
	if err != nil {
		return core.SourceResponse{}, err
	}
	var feed arxivFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return core.SourceResponse{}, err
	}
	records := make([]core.SourceRecord, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		id := fields.NormalizeArXivID(entry.ID)
		records = append(records, core.SourceRecord{
			Source:   "arxiv",
			SourceID: id,
			Title:    fields.CompactSpace(entry.Title),
			Abstract: fields.CompactSpace(entry.Summary),
			Identifiers: core.Identifiers{
				ArXivID: id,
				DOI:     fields.NormalizeDOI(entry.DOI),
			},
			Year:     yearFromPublished(entry.Published),
			URLs:     arxivLinks(entry),
			Metadata: arxivMetadata(entry),
		})
	}
	return core.SourceResponse{Records: records, RawRef: rawArXivRef(params)}, nil
}

type arxivFeed struct {
	Entries []arxivEntry `xml:"entry"`
}

type arxivEntry struct {
	ID         string          `xml:"id"`
	Title      string          `xml:"title"`
	Summary    string          `xml:"summary"`
	Published  string          `xml:"published"`
	Links      []arxivLink     `xml:"link"`
	Categories []arxivCategory `xml:"category"`
	DOI        string          `xml:"http://arxiv.org/schemas/atom doi"`
	Comment    string          `xml:"http://arxiv.org/schemas/atom comment"`
	JournalRef string          `xml:"http://arxiv.org/schemas/atom journal_ref"`
}

type arxivLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type arxivCategory struct {
	Term string `xml:"term,attr"`
}

func arxivSearchQuery(query core.SourceQuery) string {
	terms := strings.TrimSpace(query.Terms)
	category := strings.TrimSpace(query.Filters["category"])
	parts := []string{}
	if category != "" {
		parts = append(parts, "cat:"+category)
	}
	if terms != "" {
		parts = append(parts, "all:"+terms)
	}
	if len(parts) == 0 {
		return "all:"
	}
	return strings.Join(parts, " AND ")
}

func rawArXivRef(params map[string]string) string {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return "arxiv:/api/query?" + values.Encode()
}

func yearFromPublished(value string) int {
	if len(value) < 4 {
		return 0
	}
	year, err := strconv.Atoi(value[:4])
	if err != nil {
		return 0
	}
	return year
}

func arxivLinks(entry arxivEntry) []string {
	links := []string{}
	for _, link := range entry.Links {
		if strings.TrimSpace(link.Href) != "" {
			links = append(links, strings.TrimSpace(link.Href))
		}
	}
	baseID := fields.NormalizeArXivID(entry.ID)
	if baseID != "" {
		links = append(links, "https://arxiv.org/pdf/"+baseID, "https://arxiv.org/e-print/"+baseID)
	}
	if len(links) == 0 && strings.TrimSpace(entry.ID) != "" {
		links = append(links, strings.TrimSpace(entry.ID))
	}
	return fields.NonEmptyStrings(links...)
}

func arxivMetadata(entry arxivEntry) map[string]string {
	metadata := map[string]string{}
	if version := arxivVersion(entry.ID); version != "" {
		metadata["version"] = version
	}
	if comment := fields.CompactSpace(entry.Comment); comment != "" {
		metadata["comment"] = comment
	}
	if journalRef := fields.CompactSpace(entry.JournalRef); journalRef != "" {
		metadata["journal_ref"] = journalRef
	}
	categories := []string{}
	for _, category := range entry.Categories {
		term := strings.TrimSpace(category.Term)
		if term != "" {
			categories = append(categories, term)
		}
	}
	if len(categories) > 0 {
		metadata["categories"] = strings.Join(categories, ",")
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func arxivVersion(value string) string {
	value = strings.TrimSpace(value)
	idx := strings.LastIndex(value, "v")
	if idx < 0 || idx == len(value)-1 {
		return ""
	}
	for _, r := range value[idx+1:] {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return value[idx:]
}
