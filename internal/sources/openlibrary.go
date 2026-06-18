package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// OpenLibraryConnector searches the Open Library book catalog.
//
// Open Library (openlibrary.org) is the Internet Archive's open catalog of
// books, covering millions of titles from ancient to modern. The search JSON
// API requires no authentication. Records without a DOI use the Open Library
// work key as a CrossrefID. OpenAccess is set when the ebook_access field
// indicates the book is freely available online.
type OpenLibraryConnector struct {
	http HTTPClient
}

// NewOpenLibraryConnector creates an Open Library source connector.
func NewOpenLibraryConnector(httpClient HTTPClient) OpenLibraryConnector {
	return OpenLibraryConnector{http: httpClient}
}

// Name returns the connector source name.
func (OpenLibraryConnector) Name() string { return "openlibrary" }

// Search queries the Open Library search API and normalizes results into
// SourceRecords. Books rarely have DOIs; the Open Library work key (e.g.
// OL20709638W) is used as the CrossrefID fallback.
func (c OpenLibraryConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"q":      query.Terms,
		"limit":  strconv.Itoa(limit),
		"fields": "key,title,author_name,first_publish_year,isbn,publisher,subject,ebook_access",
	}
	body, err := c.http.Get(ctx, "/search.json", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload olSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Docs))
	for _, doc := range payload.Docs {
		title := strings.TrimSpace(doc.Title)
		if title == "" {
			continue
		}
		// Key is /works/OL20709638W — strip leading slash for SourceID.
		workID := strings.TrimPrefix(strings.TrimSpace(doc.Key), "/works/")
		if workID == "" {
			workID = strings.TrimPrefix(strings.TrimSpace(doc.Key), "/")
		}
		crossrefID := "openlibrary:" + workID
		year := doc.FirstPublishYear
		var authorNames []string
		for _, a := range doc.AuthorName {
			if a := strings.TrimSpace(a); a != "" {
				authorNames = append(authorNames, a)
			}
		}
		publisher := ""
		if len(doc.Publisher) > 0 {
			publisher = strings.TrimSpace(doc.Publisher[0])
		}
		isbn := ""
		if len(doc.ISBN) > 0 {
			isbn = strings.TrimSpace(doc.ISBN[0])
		}
		oa := strings.TrimSpace(doc.EbookAccess) == "public"
		metadata := map[string]string{}
		if len(authorNames) > 0 {
			metadata["authors"] = strings.Join(authorNames, "; ")
		}
		if isbn != "" {
			metadata["isbn"] = isbn
		}
		if ebook := strings.TrimSpace(doc.EbookAccess); ebook != "" {
			metadata["ebook_access"] = ebook
		}
		records = append(records, SourceRecord{
			Source:   "openlibrary",
			SourceID: workID,
			Title:    title,
			Identifiers: Identifiers{
				CrossrefID: crossrefID,
			},
			Year:       year,
			Publisher:  publisher,
			OpenAccess: oa,
			Metadata:   metadata,
		})
	}
	rawRef := fmt.Sprintf("openlibrary:/search.json?q=%s&limit=%d", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type olSearchResponse struct {
	NumFound int      `json:"numFound"`
	Start    int      `json:"start"`
	Docs     []olDoc  `json:"docs"`
}

type olDoc struct {
	Key              string   `json:"key"`
	Title            string   `json:"title"`
	AuthorName       []string `json:"author_name"`
	FirstPublishYear int      `json:"first_publish_year"`
	ISBN             []string `json:"isbn"`
	Publisher        []string `json:"publisher"`
	Subject          []string `json:"subject"`
	EbookAccess      string   `json:"ebook_access"`
}
