package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ZbMATHConnector searches the zbMATH Open mathematics database.
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
		"q":        query.Terms,
		"per_page": strconv.Itoa(limit),
	}
	body, err := c.http.Get(ctx, "/v1/document/", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload zbmathSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Result.Hits))
	for _, hit := range payload.Result.Hits {
		doi := normalizeSourceDOI(hit.DOI)
		zblID := strings.TrimSpace(hit.ZblID)

		// Collect MSC codes.
		mscParts := make([]string, 0, len(hit.MSCCodes))
		for _, m := range hit.MSCCodes {
			if code := strings.TrimSpace(m.MSCCode); code != "" {
				mscParts = append(mscParts, code)
			}
		}

		ids := Identifiers{
			DOI: doi,
		}
		// Fall back to zbmath: prefixed ZblID when no DOI is available so that
		// library validation does not reject the record.
		if doi == "" && zblID != "" {
			ids.CrossrefID = "zbmath:" + zblID
		}

		records = append(records, SourceRecord{
			Source:      "zbmath",
			SourceID:    strconv.Itoa(hit.DocumentID),
			Title:       strings.TrimSpace(hit.Title),
			Identifiers: ids,
			Year:        hit.Year,
			Abstract:    strings.TrimSpace(hit.Abstract),
			Venue:       strings.TrimSpace(hit.Journal.Name),
			URLs:        nonEmptyStrings(zbmathRecordURL(zblID)),
			Metadata: map[string]string{
				"zbl_id":    zblID,
				"msc_codes": strings.Join(mscParts, "; "),
			},
		})
	}
	return SourceResponse{Records: records, RawRef: rawZbMATHRef(query.Terms, limit)}, nil
}

type zbmathSearchResponse struct {
	Result struct {
		Hits  []zbmathHit `json:"hits"`
		Count int         `json:"count"`
	} `json:"result"`
}

type zbmathHit struct {
	DocumentID int    `json:"document_id"`
	Title      string `json:"title"`
	Abstract   string `json:"abstract"`
	Year       int    `json:"year"`
	Authors    []struct {
		Name string `json:"name"`
	} `json:"authors"`
	Journal struct {
		Name string `json:"name"`
	} `json:"journal"`
	DOI      string `json:"doi"`
	ZblID    string `json:"zbl_id"`
	MSCCodes []struct {
		MSCCode string `json:"msc_code"`
	} `json:"msc_codes"`
}

func rawZbMATHRef(terms string, limit int) string {
	return fmt.Sprintf("zbmath:/v1/document/?q=%s&per_page=%d", url.QueryEscape(terms), limit)
}

func zbmathRecordURL(zblID string) string {
	if zblID == "" {
		return ""
	}
	return "https://zbmath.org/?q=an:" + zblID
}
