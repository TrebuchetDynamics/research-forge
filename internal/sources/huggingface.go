package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// HuggingFaceConnector searches the HuggingFace Papers index.
//
// HuggingFace Papers (huggingface.co/papers) is a community-curated feed of
// machine learning and AI research papers. Each paper corresponds to an arXiv
// preprint; the HuggingFace index adds community upvotes, code repository
// links, and AI-generated summaries. The JSON API requires no authentication.
type HuggingFaceConnector struct {
	http HTTPClient
}

// NewHuggingFaceConnector creates a HuggingFace Papers source connector.
func NewHuggingFaceConnector(httpClient HTTPClient) HuggingFaceConnector {
	return HuggingFaceConnector{http: httpClient}
}

// Name returns the connector source name.
func (HuggingFaceConnector) Name() string { return "huggingface" }

// Search queries the HuggingFace Papers API and normalizes results into
// SourceRecords. Papers are arXiv preprints; DOIs follow the arXiv DOI scheme
// (10.48550/arXiv.<id>).
func (c HuggingFaceConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"q":     query.Terms,
		"limit": strconv.Itoa(limit),
	}
	body, err := c.http.Get(ctx, "/api/papers", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var papers []hfPaper
	if err := json.Unmarshal(body, &papers); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(papers))
	for _, p := range papers {
		title := strings.TrimSpace(p.Title)
		if title == "" {
			continue
		}
		arxivID := strings.TrimSpace(p.ID)
		// Construct the official arXiv DOI (10.48550/arXiv.<id>) and
		// normalize it — the lower-cased form matches CrossRef normalization.
		doi := ""
		if arxivID != "" {
			doi = normalizeSourceDOI("10.48550/arXiv." + arxivID)
		}
		crossrefID := ""
		if doi == "" {
			crossrefID = "huggingface:" + arxivID
		}
		year := 0
		if len(p.PublishedAt) >= 4 {
			year, _ = strconv.Atoi(p.PublishedAt[:4])
		}
		var authorNames []string
		for _, a := range p.Authors {
			if a.Name != "" {
				authorNames = append(authorNames, a.Name)
			}
		}
		metadata := map[string]string{}
		if len(authorNames) > 0 {
			metadata["authors"] = strings.Join(authorNames, "; ")
		}
		if p.Upvotes > 0 {
			metadata["upvotes"] = strconv.Itoa(p.Upvotes)
		}
		if p.GitHubRepo != "" {
			metadata["github_repo"] = p.GitHubRepo
		}
		var urls []string
		if arxivID != "" {
			urls = append(urls, "https://arxiv.org/abs/"+arxivID)
		}
		records = append(records, SourceRecord{
			Source:   "huggingface",
			SourceID: arxivID,
			Title:    title,
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:       year,
			Abstract:   strings.TrimSpace(p.Summary),
			OpenAccess: true,
			URLs:       nonEmptyStrings(urls...),
			Metadata:   metadata,
		})
	}
	rawRef := fmt.Sprintf("huggingface:/api/papers?q=%s&limit=%d", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type hfPaper struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	PublishedAt string     `json:"publishedAt"`
	Authors     []hfAuthor `json:"authors"`
	Summary     string     `json:"summary"`
	Upvotes     int        `json:"upvotes"`
	GitHubRepo  string     `json:"githubRepo"`
}

type hfAuthor struct {
	Name string `json:"name"`
}
