package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/core"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/fields"
	"github.com/TrebuchetDynamics/research-forge/internal/sources/transport"
	"net/url"
	"strconv"
	"strings"
)

// SemanticScholarConnector searches the Semantic Scholar Graph API paper index.
type SemanticScholarConnector struct {
	http transport.HTTPClient
}

// NewSemanticScholarConnector creates a Semantic Scholar source connector.
func NewSemanticScholarConnector(httpClient transport.HTTPClient) SemanticScholarConnector {
	return SemanticScholarConnector{http: httpClient}
}

// Name returns the connector source name.
func (SemanticScholarConnector) Name() string { return "semantic-scholar" }

// Search queries Semantic Scholar papers and normalizes results into SourceRecords.
func (c SemanticScholarConnector) Search(ctx context.Context, query core.SourceQuery) (core.SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{
		"query":  query.Terms,
		"limit":  strconv.Itoa(limit),
		"fields": "paperId,title,abstract,year,publicationDate,venue,url,isOpenAccess,openAccessPdf,externalIds",
	}
	if cap := strings.TrimSpace(query.Filters["publicationDateOrYear"]); cap != "" {
		params["publicationDateOrYear"] = cap
	}
	if strings.TrimSpace(query.PageCursor) != "" {
		params["offset"] = strings.TrimSpace(query.PageCursor)
	}
	body, err := c.http.Get(ctx, "/graph/v1/paper/search", params)
	if err != nil {
		return core.SourceResponse{}, err
	}
	var payload semanticScholarSearchResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return core.SourceResponse{}, err
	}
	records := make([]core.SourceRecord, 0, len(payload.Data))
	for _, paper := range payload.Data {
		paperID := strings.TrimSpace(paper.PaperID)
		records = append(records, core.SourceRecord{
			Source:   "semantic-scholar",
			SourceID: paperID,
			Title:    strings.TrimSpace(paper.Title),
			Identifiers: core.Identifiers{
				DOI:               fields.NormalizeDOI(paper.ExternalIDs.DOI),
				ArXivID:           strings.TrimPrefix(strings.TrimSpace(paper.ExternalIDs.ArXiv), "arXiv:"),
				SemanticScholarID: paperID,
			},
			Year:       paper.Year,
			Abstract:   strings.TrimSpace(paper.Abstract),
			Venue:      strings.TrimSpace(paper.Venue),
			URLs:       fields.NonEmptyStrings(paper.URL, paper.OpenAccessPDF.URL),
			OpenAccess: paper.IsOpenAccess,
			Metadata: map[string]string{
				"pubmed_id":         strings.TrimSpace(paper.ExternalIDs.PubMed),
				"publication_dates": strings.TrimSpace(paper.PublicationDate),
			},
		})
	}
	return core.SourceResponse{Records: records, RawRef: rawSemanticScholarRef(params), NextPageCursor: nextSemanticScholarCursor(payload.Next)}, nil
}

type semanticScholarSearchResponse struct {
	Next *int                   `json:"next"`
	Data []semanticScholarPaper `json:"data"`
}

type semanticScholarPaper struct {
	PaperID         string                       `json:"paperId"`
	Title           string                       `json:"title"`
	Abstract        string                       `json:"abstract"`
	Year            int                          `json:"year"`
	PublicationDate string                       `json:"publicationDate"`
	Venue           string                       `json:"venue"`
	URL             string                       `json:"url"`
	IsOpenAccess    bool                         `json:"isOpenAccess"`
	OpenAccessPDF   semanticScholarOpenAccessPDF `json:"openAccessPdf"`
	ExternalIDs     semanticScholarExternalIDs   `json:"externalIds"`
}

type semanticScholarOpenAccessPDF struct {
	URL string `json:"url"`
}

type semanticScholarExternalIDs struct {
	DOI    string `json:"DOI"`
	ArXiv  string `json:"ArXiv"`
	PubMed string `json:"PubMed"`
}

func rawSemanticScholarRef(params map[string]string) string {
	values := url.Values{}
	for _, key := range []string{"limit", "offset", "publicationDateOrYear", "query"} {
		if value := strings.TrimSpace(params[key]); value != "" {
			values.Set(key, value)
		}
	}
	return fmt.Sprintf("semantic-scholar:/graph/v1/paper/search?%s", values.Encode())
}

// ExpandCitationGraph fetches references, citations, or both around one Semantic Scholar paper.
func (c SemanticScholarConnector) ExpandCitationGraph(ctx context.Context, query core.SemanticScholarGraphQuery) (core.CitationGraphExpansion, error) {
	paperID := strings.TrimSpace(query.PaperID)
	if paperID == "" {
		return core.CitationGraphExpansion{}, fmt.Errorf("semantic scholar paper id is required")
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	direction := query.Direction
	if direction == "" {
		direction = core.SemanticScholarDirectionBoth
	}
	expansion := core.CitationGraphExpansion{SeedID: paperID, Records: map[string]core.SourceRecord{}}
	params := map[string]string{"limit": strconv.Itoa(limit), "fields": "paperId,title,abstract,year,venue,url,isOpenAccess,openAccessPdf,externalIds"}
	if direction == core.SemanticScholarDirectionReferences || direction == core.SemanticScholarDirectionBoth {
		refs, err := c.semanticScholarGraphPage(ctx, paperID, "references", params)
		if err != nil {
			return core.CitationGraphExpansion{}, err
		}
		for _, item := range refs.Data {
			record := sourceRecordFromSemanticScholarPaper(item.CitedPaper)
			if record.SourceID == "" {
				continue
			}
			expansion.Edges = append(expansion.Edges, core.CitationEdge{SourceID: paperID, TargetID: record.SourceID})
			expansion.Records[record.SourceID] = record
		}
	}
	if direction == core.SemanticScholarDirectionCitations || direction == core.SemanticScholarDirectionBoth {
		cites, err := c.semanticScholarGraphPage(ctx, paperID, "citations", params)
		if err != nil {
			return core.CitationGraphExpansion{}, err
		}
		for _, item := range cites.Data {
			record := sourceRecordFromSemanticScholarPaper(item.CitingPaper)
			if record.SourceID == "" {
				continue
			}
			expansion.Edges = append(expansion.Edges, core.CitationEdge{SourceID: record.SourceID, TargetID: paperID})
			expansion.Records[record.SourceID] = record
		}
	}
	expansion.RawRef = rawSemanticScholarGraphRef(paperID, direction, limit)
	return expansion, nil
}

func (c SemanticScholarConnector) semanticScholarGraphPage(ctx context.Context, paperID, relation string, params map[string]string) (semanticScholarGraphResponse, error) {
	body, err := c.http.Get(ctx, "/graph/v1/paper/"+url.PathEscape(paperID)+"/"+relation, params)
	if err != nil {
		return semanticScholarGraphResponse{}, err
	}
	var payload semanticScholarGraphResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return semanticScholarGraphResponse{}, err
	}
	return payload, nil
}

type semanticScholarGraphResponse struct {
	Data []semanticScholarGraphItem `json:"data"`
}

type semanticScholarGraphItem struct {
	CitedPaper  semanticScholarPaper `json:"citedPaper"`
	CitingPaper semanticScholarPaper `json:"citingPaper"`
}

func sourceRecordFromSemanticScholarPaper(paper semanticScholarPaper) core.SourceRecord {
	paperID := strings.TrimSpace(paper.PaperID)
	return core.SourceRecord{
		Source:   "semantic-scholar",
		SourceID: paperID,
		Title:    strings.TrimSpace(paper.Title),
		Identifiers: core.Identifiers{
			DOI:               fields.NormalizeDOI(paper.ExternalIDs.DOI),
			ArXivID:           strings.TrimPrefix(strings.TrimSpace(paper.ExternalIDs.ArXiv), "arXiv:"),
			SemanticScholarID: paperID,
		},
		Year:       paper.Year,
		Abstract:   strings.TrimSpace(paper.Abstract),
		Venue:      strings.TrimSpace(paper.Venue),
		URLs:       fields.NonEmptyStrings(paper.URL, paper.OpenAccessPDF.URL),
		OpenAccess: paper.IsOpenAccess,
		Metadata: map[string]string{
			"pubmed_id": strings.TrimSpace(paper.ExternalIDs.PubMed),
		},
	}
}

func rawSemanticScholarGraphRef(paperID string, direction core.SemanticScholarGraphDirection, limit int) string {
	relation := string(direction)
	if direction == core.SemanticScholarDirectionBoth {
		relation = "references+citations"
	}
	return fmt.Sprintf("semantic-scholar:/graph/v1/paper/%s/%s?limit=%d", paperID, relation, limit)
}

func nextSemanticScholarCursor(next *int) string {
	if next == nil {
		return ""
	}
	return strconv.Itoa(*next)
}
