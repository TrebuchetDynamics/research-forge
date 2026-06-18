package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const zbmathLicenseMsg = "zbMATH Open Web Interface contents unavailable due to conflicting licenses."

// ZbMATHConnector searches the zbMATH Open mathematics database.
//
// Some records have title/abstract redacted due to publisher license conflicts;
// those records are skipped. The search path is /v1/document/_search with
// search_string parameter (not /v1/document/ with q=).
type ZbMATHConnector struct {
	http HTTPClient
}

// NewZbMATHConnector creates a zbMATH source connector.
func NewZbMATHConnector(httpClient HTTPClient) ZbMATHConnector {
	return ZbMATHConnector{http: httpClient}
}

// Name returns the connector source name.
func (ZbMATHConnector) Name() string { return "zbmath" }

// Search queries the zbMATH Open REST API and normalizes results into SourceRecords.
func (c ZbMATHConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"search_string": query.Terms,
		"per_page":      strconv.Itoa(limit),
	}
	body, err := c.http.Get(ctx, "/v1/document/_search", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload zbmathSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Result))
	for _, hit := range payload.Result {
		title := strings.TrimSpace(hit.Title.Title)
		if title == "" || title == zbmathLicenseMsg {
			continue
		}
		doi := ""
		for _, link := range hit.Links {
			if link.Type == "doi" {
				doi = normalizeSourceDOI(link.Identifier)
				break
			}
		}
		zblID := strings.TrimSpace(hit.Identifier)
		venue := ""
		if len(hit.Source.Series) > 0 {
			venue = strings.TrimSpace(hit.Source.Series[0].Title)
		}
		mscParts := make([]string, 0, len(hit.MSC))
		for _, m := range hit.MSC {
			if code := strings.TrimSpace(m.Code); code != "" {
				mscParts = append(mscParts, code)
			}
		}
		year := 0
		if raw := strings.Trim(string(hit.Year), `"`); raw != "" && raw != "null" {
			year, _ = strconv.Atoi(raw)
		}
		ids := Identifiers{DOI: doi}
		if doi == "" && zblID != "" {
			ids.CrossrefID = "zbmath:" + zblID
		}
		records = append(records, SourceRecord{
			Source:      "zbmath",
			SourceID:    strconv.Itoa(hit.ID),
			Title:       title,
			Identifiers: ids,
			Year:        year,
			Venue:       venue,
			URLs:        nonEmptyStrings(hit.ZbMATHURL),
			Metadata: map[string]string{
				"zbl_id":    zblID,
				"msc_codes": strings.Join(mscParts, "; "),
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawZbMATHRef(query.Terms, limit)}, nil
}

type zbmathSearchResponse struct {
	Result []zbmathHit `json:"result"`
}

type zbmathHit struct {
	ID    int `json:"id"`
	Title struct {
		Title string `json:"title"`
	} `json:"title"`
	Year       json.RawMessage `json:"year"` // API returns year as string e.g. "1992"
	Identifier string          `json:"identifier"`
	ZbMATHURL  string `json:"zbmath_url"`
	Links      []struct {
		Identifier string `json:"identifier"`
		Type       string `json:"type"`
		URL        string `json:"url"`
	} `json:"links"`
	Source struct {
		Series []struct {
			Title string `json:"title"`
		} `json:"series"`
	} `json:"source"`
	MSC []struct {
		Code string `json:"code"`
	} `json:"msc"`
}

func rawZbMATHRef(terms string, limit int) string {
	return fmt.Sprintf("zbmath:/v1/document/_search?search_string=%s&per_page=%d", url.QueryEscape(terms), limit)
}
