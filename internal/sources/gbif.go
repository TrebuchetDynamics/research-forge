package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// GBIFConnector searches the GBIF Literature API.
//
// GBIF (Global Biodiversity Information Facility) maintains a curated
// bibliography of literature that cites GBIF data or covers biodiversity
// informatics topics. The public REST API requires no authentication.
type GBIFConnector struct {
	http HTTPClient
}

// NewGBIFConnector creates a GBIF Literature source connector.
func NewGBIFConnector(httpClient HTTPClient) GBIFConnector {
	return GBIFConnector{http: httpClient}
}

// Name returns the connector source name.
func (GBIFConnector) Name() string { return "gbif" }

// Search queries the GBIF Literature API and normalizes results into SourceRecords.
func (c GBIFConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"q":      query.Terms,
		"limit":  strconv.Itoa(limit),
		"offset": "0",
	}
	body, err := c.http.Get(ctx, "/v1/literature/search", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload gbifLiteratureResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Results))
	for _, lit := range payload.Results {
		title := strings.TrimSpace(lit.Title)
		if title == "" {
			continue
		}
		doi := normalizeSourceDOI(lit.Identifiers.DOI)
		crossrefID := ""
		if doi == "" {
			crossrefID = "gbif:" + lit.ID
		}
		// Build author list as "LastName, FirstName".
		var authorParts []string
		for _, a := range lit.Authors {
			name := strings.TrimSpace(a.LastName + ", " + a.FirstName)
			name = strings.TrimSuffix(strings.TrimPrefix(name, ", "), ", ")
			if name != "" && name != "," {
				authorParts = append(authorParts, name)
			}
		}
		metadata := map[string]string{}
		if len(authorParts) > 0 {
			metadata["authors"] = strings.Join(authorParts, "; ")
		}
		if lit.LitType != "" {
			metadata["literature_type"] = lit.LitType
		}
		if lit.PeerReview {
			metadata["peer_review"] = "true"
		}
		websiteURL := ""
		if len(lit.Websites) > 0 {
			websiteURL = lit.Websites[0]
		}
		records = append(records, SourceRecord{
			Source:   "gbif",
			SourceID: lit.ID,
			Title:    title,
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:       lit.Year,
			Abstract:   lit.Abstract,
			Venue:      lit.Source,
			OpenAccess: lit.OpenAccess,
			URLs:       nonEmptyStrings(websiteURL),
			Metadata:   metadata,
		})
	}
	rawRef := fmt.Sprintf("gbif:/v1/literature/search?q=%s&limit=%d&offset=0", url.QueryEscape(query.Terms), limit)
	return SourceResponse{Records: records, RawRef: rawRef}, nil
}

type gbifLiteratureResponse struct {
	Offset       int            `json:"offset"`
	Limit        int            `json:"limit"`
	EndOfRecords bool           `json:"endOfRecords"`
	Count        int            `json:"count"`
	Results      []gbifLitEntry `json:"results"`
}

type gbifLitEntry struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Authors     []gbifAuthor    `json:"authors"`
	Identifiers gbifIdentifiers `json:"identifiers"`
	Year        int             `json:"year"`
	Abstract    string          `json:"abstract"`
	Source      string          `json:"source"`
	LitType     string          `json:"literatureType"`
	OpenAccess  bool            `json:"openAccess"`
	PeerReview  bool            `json:"peerReview"`
	Websites    []string        `json:"websites"`
}

type gbifAuthor struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type gbifIdentifiers struct {
	DOI string `json:"doi"`
}
