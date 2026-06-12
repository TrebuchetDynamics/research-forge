package sources

import (
	"context"
	"encoding/xml"
	"net/url"
	"strconv"
	"strings"
)

// ArXivConnector searches arXiv Atom feeds.
type ArXivConnector struct {
	http HTTPClient
}

// NewArXivConnector creates an arXiv source connector.
func NewArXivConnector(httpClient HTTPClient) ArXivConnector {
	return ArXivConnector{http: httpClient}
}

// Name returns the connector source name.
func (ArXivConnector) Name() string { return "arxiv" }

// Search queries arXiv and normalizes entries into SourceRecords.
func (c ArXivConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{"search_query": "all:" + query.Terms, "max_results": strconv.Itoa(limit)}
	body, err := c.http.Get(ctx, "/api/query", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var feed arxivFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		id := normalizeArXivID(entry.ID)
		records = append(records, SourceRecord{
			Source:   "arxiv",
			SourceID: id,
			Title:    compactSpace(entry.Title),
			Abstract: compactSpace(entry.Summary),
			Identifiers: Identifiers{
				ArXivID: id,
			},
			Year:     yearFromPublished(entry.Published),
			URLs:     arxivLinks(entry),
			Metadata: arxivMetadata(entry),
		})
	}
	return SourceResponse{Records: records, RawRef: rawArXivRef(params)}, nil
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
}

type arxivLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type arxivCategory struct {
	Term string `xml:"term,attr"`
}

func rawArXivRef(params map[string]string) string {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return "arxiv:/api/query?" + values.Encode()
}

func normalizeArXivID(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "http://arxiv.org/abs/")
	value = strings.TrimPrefix(value, "https://arxiv.org/abs/")
	if idx := strings.Index(value, "v"); idx > 0 {
		value = value[:idx]
	}
	return value
}

func compactSpace(value string) string {
	return strings.Join(strings.Fields(value), " ")
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
	if len(links) == 0 && strings.TrimSpace(entry.ID) != "" {
		links = append(links, strings.TrimSpace(entry.ID))
	}
	return links
}

func arxivMetadata(entry arxivEntry) map[string]string {
	metadata := map[string]string{}
	if version := arxivVersion(entry.ID); version != "" {
		metadata["version"] = version
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
