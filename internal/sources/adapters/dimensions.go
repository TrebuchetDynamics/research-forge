package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/core"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/fields"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/transport"
	"strings"
)

// DimensionsConnector searches Dimensions (free academic API tier).
//
// Dimensions requires a JWT bearer token obtained via the free academic access
// program at app.dimensions.ai. Set the RFORGE_DIMENSIONS_TOKEN environment
// variable to enable this connector. The connector uses the DSL POST endpoint
// with a plain-text query body.
//
// DSL reference: https://docs.dimensions.ai/dsl/
type DimensionsConnector struct {
	http transport.HTTPClient
}

// NewDimensionsConnector creates a Dimensions source connector.
func NewDimensionsConnector(httpClient transport.HTTPClient) DimensionsConnector {
	return DimensionsConnector{http: httpClient}
}

// Name returns the connector source name.
func (DimensionsConnector) Name() string { return "dimensions" }

// Search queries the Dimensions DSL API and normalizes results into SourceRecords.
func (c DimensionsConnector) Search(ctx context.Context, query core.SourceQuery) (core.SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	// Escape double quotes in query to avoid breaking the DSL string literal.
	safeQuery := strings.ReplaceAll(query.Terms, `"`, `'`)
	dsl := fmt.Sprintf(
		`search publications in title_abstract_only for "%s" return publications[id+doi+title+year+journal.title+abstract+open_access] limit %d`,
		safeQuery, limit,
	)
	body, err := c.http.PostText(ctx, "/api/dsl.json", []byte(dsl))
	if err != nil {
		return core.SourceResponse{}, err
	}
	var payload dimensionsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return core.SourceResponse{}, err
	}
	records := make([]core.SourceRecord, 0, len(payload.Publications))
	for _, pub := range payload.Publications {
		title := strings.TrimSpace(pub.Title)
		if title == "" {
			continue
		}
		doi := fields.NormalizeDOI(pub.DOI)
		ids := core.Identifiers{DOI: doi}
		if doi == "" && pub.ID != "" {
			ids.CrossrefID = "dimensions:" + pub.ID
		}
		journal := ""
		if pub.Journal != nil {
			journal = strings.TrimSpace(pub.Journal.Title)
		}
		records = append(records, core.SourceRecord{
			Source:      "dimensions",
			SourceID:    pub.ID,
			Title:       title,
			Identifiers: ids,
			Year:        pub.Year,
			Abstract:    strings.TrimSpace(pub.Abstract),
			Venue:       journal,
			OpenAccess:  pub.OpenAccess,
			URLs:        fields.NonEmptyStrings(fields.DOIURL(doi)),
			Metadata:    map[string]string{"dimensions_id": pub.ID},
		})
	}
	return core.SourceResponse{Records: records, RawRef: "dimensions:/api/dsl.json"}, nil
}

type dimensionsResponse struct {
	Publications []dimensionsPublication `json:"publications"`
}

type dimensionsPublication struct {
	ID      string `json:"id"`
	DOI     string `json:"doi"`
	Title   string `json:"title"`
	Year    int    `json:"year"`
	Journal *struct {
		Title string `json:"title"`
	} `json:"journal"`
	Abstract   string `json:"abstract"`
	OpenAccess bool   `json:"open_access"`
}
