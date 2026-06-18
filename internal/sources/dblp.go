package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DBLPConnector searches the DBLP computer science bibliography.
type DBLPConnector struct {
	http HTTPClient
}

// NewDBLPConnector creates a DBLP source connector.
func NewDBLPConnector(httpClient HTTPClient) DBLPConnector {
	return DBLPConnector{http: httpClient}
}

// Name returns the connector source name.
func (DBLPConnector) Name() string { return "dblp" }

// Search queries the DBLP publication search API and normalizes results into SourceRecords.
func (c DBLPConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"q":      query.Terms,
		"format": "json",
		"h":      strconv.Itoa(limit),
		"f":      "0",
	}
	body, err := c.http.Get(ctx, "/search/publ/api", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload dblpSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Result.Hits.Hit))
	for _, hit := range payload.Result.Hits.Hit {
		info := hit.Info
		doi := normalizeSourceDOI(strings.TrimSpace(info.DOI))
		ee := strings.TrimSpace(info.EE)
		htmlURL := strings.TrimSpace(info.URL)
		urls := nonEmptyStrings(ee, htmlURL)
		year, _ := strconv.Atoi(strings.TrimSpace(info.Year))
		records = append(records, SourceRecord{
			Source:   "dblp",
			SourceID: strings.TrimSpace(hit.ID),
			Title:    strings.TrimSpace(info.Title),
			Identifiers: Identifiers{
				DOI: doi,
			},
			Year:  year,
			Venue: strings.TrimSpace(info.Venue),
			URLs:  urls,
			Metadata: map[string]string{
				"authors": dblpAuthorNames(info.Authors),
				"ee":      ee,
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawDBLPRef(params)}, nil
}

func dblpAuthorNames(authors dblpAuthors) string {
	names := make([]string, 0, len(authors.Author))
	for _, a := range authors.Author {
		if name := strings.TrimSpace(a.Text); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, "; ")
}

type dblpSearchResponse struct {
	Result struct {
		Hits struct {
			Total string    `json:"@total"`
			Hit   []dblpHit `json:"hit"`
		} `json:"hits"`
	} `json:"result"`
}

type dblpHit struct {
	ID   string   `json:"@id"`
	Info dblpInfo `json:"info"`
}

type dblpInfo struct {
	Title   string      `json:"title"`
	Authors dblpAuthors `json:"authors"`
	Year    string      `json:"year"`
	Venue   string      `json:"venue"`
	DOI     string      `json:"doi"`
	URL     string      `json:"url"`
	EE      string      `json:"ee"`
}

// dblpAuthors handles both array and single-object author fields from the DBLP API.
type dblpAuthors struct {
	Author []dblpAuthor
}

type dblpAuthor struct {
	Text string
}

func (a *dblpAuthors) UnmarshalJSON(data []byte) error {
	var raw struct {
		Author json.RawMessage `json:"author"`
	}
	if err := json.Unmarshal(data, &raw); err != nil || raw.Author == nil {
		return nil
	}
	// Try array first, then single object.
	var arr []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw.Author, &arr); err == nil {
		for _, item := range arr {
			a.Author = append(a.Author, dblpAuthor{Text: item.Text})
		}
		return nil
	}
	var single struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw.Author, &single); err == nil {
		a.Author = append(a.Author, dblpAuthor{Text: single.Text})
	}
	return nil
}

func rawDBLPRef(params map[string]string) string {
	values := url.Values{}
	for _, key := range []string{"f", "format", "h", "q"} {
		if v := strings.TrimSpace(params[key]); v != "" {
			values.Set(key, v)
		}
	}
	return fmt.Sprintf("dblp:/search/publ/api?%s", values.Encode())
}
