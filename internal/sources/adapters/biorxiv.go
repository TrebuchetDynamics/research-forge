package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/core"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/fields"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/transport"
	"strings"
	"time"
)

// BioRxivConnector lists recent preprints from bioRxiv or medRxiv.
//
// The biorxiv.org API does not support keyword search — it lists preprints
// by date range. This connector fetches the most recent 30 days of preprints
// from the requested server and filters by the query terms against title and
// abstract. For keyword-based discovery of bioRxiv content, the europepmc
// and semantic-scholar connectors provide broader coverage.
//
// Use the filter "server=medrxiv" to search medRxiv instead.
type BioRxivConnector struct {
	http transport.HTTPClient
	now  func() time.Time
}

// NewBioRxivConnector creates a bioRxiv/medRxiv source connector.
func NewBioRxivConnector(httpClient transport.HTTPClient) BioRxivConnector {
	return BioRxivConnector{http: httpClient, now: time.Now}
}

// newBioRxivConnectorWithClock creates a bioRxiv connector with an injected clock for testing.
func newBioRxivConnectorWithClock(httpClient transport.HTTPClient, now func() time.Time) BioRxivConnector {
	return BioRxivConnector{http: httpClient, now: now}
}

// Name returns the connector source name.
func (BioRxivConnector) Name() string { return "biorxiv" }

// Search lists recent preprints from bioRxiv (or medRxiv via filter) and
// filters by query terms against title and abstract.
func (c BioRxivConnector) Search(ctx context.Context, query core.SourceQuery) (core.SourceResponse, error) {
	server := "biorxiv"
	if s := strings.TrimSpace(query.Filters["server"]); s == "medrxiv" {
		server = "medrxiv"
	}
	today := c.now().UTC()
	start := today.AddDate(0, 0, -30)
	interval := fmt.Sprintf("%s/%s", start.Format("2006-01-02"), today.Format("2006-01-02"))
	path := fmt.Sprintf("/details/%s/%s/0/json", server, interval)
	body, err := c.http.Get(ctx, path, nil)
	if err != nil {
		return core.SourceResponse{}, err
	}
	var payload bioRxivResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return core.SourceResponse{}, err
	}
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query.Terms)))
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	records := make([]core.SourceRecord, 0)
	for _, item := range payload.Collection {
		if !bioRxivMatchesTerms(item, terms) {
			continue
		}
		doi := fields.NormalizeDOI(strings.TrimSpace(item.DOI))
		year := 0
		if len(strings.TrimSpace(item.Date)) >= 4 {
			year, _ = parseInt4(strings.TrimSpace(item.Date)[:4])
		}
		htmlURL := ""
		if doi != "" {
			htmlURL = "https://doi.org/" + doi
		}
		records = append(records, core.SourceRecord{
			Source:   "biorxiv",
			SourceID: strings.TrimSpace(item.DOI),
			Title:    strings.TrimSpace(item.Title),
			Identifiers: core.Identifiers{
				DOI: doi,
			},
			Year:       year,
			Abstract:   strings.TrimSpace(item.Abstract),
			Venue:      strings.TrimSpace(item.Server),
			URLs:       fields.NonEmptyStrings(htmlURL),
			OpenAccess: true,
			Metadata: map[string]string{
				"category": strings.TrimSpace(item.Category),
				"server":   strings.TrimSpace(item.Server),
				"version":  strings.TrimSpace(item.Version),
				"authors":  strings.TrimSpace(item.Authors),
			},
		})
		if len(records) >= limit {
			break
		}
	}
	rawRef := fmt.Sprintf("biorxiv:/details/%s/%s/0/json", server, interval)
	return core.SourceResponse{Records: records, RawRef: rawRef}, nil
}

func bioRxivMatchesTerms(item bioRxivPreprint, terms []string) bool {
	if len(terms) == 0 {
		return true
	}
	haystack := strings.ToLower(item.Title + " " + item.Abstract)
	for _, term := range terms {
		if !strings.Contains(haystack, term) {
			return false
		}
	}
	return true
}

func parseInt4(s string) (int, error) {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("not a digit: %c", ch)
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}

type bioRxivResponse struct {
	Messages   []bioRxivMessage  `json:"messages"`
	Collection []bioRxivPreprint `json:"collection"`
}

type bioRxivMessage struct {
	Status   string          `json:"status"`
	Total    json.RawMessage `json:"total"`  // API returns int or string depending on version
	Cursor   json.RawMessage `json:"cursor"` // same
	Count    json.RawMessage `json:"count"`  // same
	Interval string          `json:"interval"`
}

type bioRxivPreprint struct {
	DOI      string `json:"doi"`
	Title    string `json:"title"`
	Authors  string `json:"authors"`
	Date     string `json:"date"`
	Version  string `json:"version"`
	Category string `json:"category"`
	Abstract string `json:"abstract"`
	Server   string `json:"server"`
}
