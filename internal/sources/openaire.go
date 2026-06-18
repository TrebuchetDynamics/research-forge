package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// OpenAIREConnector searches the OpenAIRE Research Graph.
type OpenAIREConnector struct {
	http HTTPClient
}

// NewOpenAIREConnector creates an OpenAIRE source connector.
func NewOpenAIREConnector(httpClient HTTPClient) OpenAIREConnector {
	return OpenAIREConnector{http: httpClient}
}

// Name returns the connector source name.
func (OpenAIREConnector) Name() string { return "openaire" }

// Search queries the OpenAIRE publications search API and normalizes results into SourceRecords.
func (c OpenAIREConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"keywords": query.Terms,
		"size":     strconv.Itoa(limit),
		"format":   "json",
		"page":     "1",
	}
	body, err := c.http.Get(ctx, "/search/publications", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload openAIRESearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	results := payload.Response.Results.Result
	records := make([]SourceRecord, 0, len(results))
	for _, result := range results {
		oafResult := result.Metadata.OAFEntity.OAFResult

		title := extractOpenAIREMainTitle(oafResult.Title)
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}

		objIdentifier := ""
		if result.Header.ObjIdentifier != nil {
			objIdentifier = result.Header.ObjIdentifier.Value
		}

		doi := extractOpenAIREDOI(oafResult.PID)
		doi = normalizeSourceDOI(doi)

		crossrefID := ""
		if doi == "" && objIdentifier != "" {
			crossrefID = "openaire:" + objIdentifier
		}

		year := 0
		if oafResult.DateOfAcceptance != nil && len(oafResult.DateOfAcceptance.Value) >= 4 {
			year, _ = strconv.Atoi(oafResult.DateOfAcceptance.Value[:4])
		}

		abstract := ""
		if oafResult.Description != nil {
			abstract = strings.TrimSpace(oafResult.Description.Value)
		}

		venue := ""
		if oafResult.Journal != nil {
			venue = strings.TrimSpace(oafResult.Journal.Title)
		}

		openAccess := false
		if oafResult.BestAccessRight != nil {
			openAccess = oafResult.BestAccessRight.ClassName == "Open Access"
		}

		urls := nonEmptyStrings(doiURL(doi))

		records = append(records, SourceRecord{
			Source:   "openaire",
			SourceID: objIdentifier,
			Title:    title,
			Identifiers: Identifiers{
				DOI:        doi,
				CrossrefID: crossrefID,
			},
			Year:       year,
			Abstract:   abstract,
			Venue:      venue,
			OpenAccess: openAccess,
			URLs:       urls,
			Metadata:   map[string]string{},
		})
	}
	return SourceResponse{
		Records: records,
		RawRef:  fmt.Sprintf("openaire:/search/publications?keywords=%s&size=%d", url.QueryEscape(query.Terms), limit),
	}, nil
}

// openAIRETextNode is a JSON node that carries only a text value field.
type openAIRETextNode struct {
	Value string `json:"$"`
}

// openAIREClassNode is a JSON node with classification attributes and a text value.
type openAIREClassNode struct {
	ClassID   string `json:"@classid"`
	ClassName string `json:"@classname"`
	Value     string `json:"$"`
}

type openAIRESearchResponse struct {
	Response struct {
		Results struct {
			Result []openAIREResult `json:"result"`
		} `json:"results"`
	} `json:"response"`
}

type openAIREResult struct {
	Header struct {
		ObjIdentifier *openAIRETextNode `json:"dri:objIdentifier"`
	} `json:"header"`
	Metadata struct {
		OAFEntity struct {
			OAFResult openAIREResultData `json:"oaf:result"`
		} `json:"oaf:entity"`
	} `json:"metadata"`
}

type openAIREResultData struct {
	Title            json.RawMessage    `json:"title"`
	PID              json.RawMessage    `json:"pid"`
	DateOfAcceptance *openAIRETextNode  `json:"dateofacceptance"`
	Description      *openAIRETextNode  `json:"description"`
	BestAccessRight  *openAIREClassNode `json:"bestaccessright"`
	Journal          *struct {
		Title string `json:"$"`
	} `json:"journal"`
	OriginalID json.RawMessage `json:"originalId"`
	Creator    json.RawMessage `json:"creator"`
}

// extractOpenAIREMainTitle handles title as list-or-single object.
// It returns the $ value of the first entry where @classid == "main title",
// falling back to the first title of any classid.
func extractOpenAIREMainTitle(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// Try as array first.
	var list []openAIREClassNode
	if err := json.Unmarshal(raw, &list); err == nil {
		fallback := ""
		for _, node := range list {
			if fallback == "" {
				fallback = node.Value
			}
			if node.ClassID == "main title" {
				return node.Value
			}
		}
		return fallback
	}
	// Try as single object.
	var single openAIREClassNode
	if err := json.Unmarshal(raw, &single); err == nil {
		return single.Value
	}
	return ""
}

// extractOpenAIREDOI handles pid as single-or-list object.
// It returns the $ value of the entry where @classid == "doi".
func extractOpenAIREDOI(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// Try as single object first.
	var single openAIREClassNode
	if err := json.Unmarshal(raw, &single); err == nil && single.ClassID != "" {
		if single.ClassID == "doi" {
			return single.Value
		}
		return ""
	}
	// Try as array.
	var list []openAIREClassNode
	if err := json.Unmarshal(raw, &list); err == nil {
		for _, node := range list {
			if node.ClassID == "doi" {
				return node.Value
			}
		}
	}
	return ""
}
