package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// OpenAlexConnector searches OpenAlex works.
type OpenAlexConnector struct {
	http HTTPClient
}

// OpenAlexEntity is a normalized OpenAlex author or institution search result.
type OpenAlexEntity struct {
	Source      string            `json:"source"`
	SourceID    string            `json:"sourceId"`
	DisplayName string            `json:"displayName"`
	WorksCount  int               `json:"worksCount"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewOpenAlexConnector creates an OpenAlex source connector.
func NewOpenAlexConnector(httpClient HTTPClient) OpenAlexConnector {
	return OpenAlexConnector{http: httpClient}
}

// Name returns the connector source name.
func (OpenAlexConnector) Name() string { return "openalex" }

// Search queries OpenAlex works and normalizes results into SourceRecords.
func (c OpenAlexConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{"per-page": strconv.Itoa(limit)}
	if strings.TrimSpace(query.Terms) != "" {
		params["search"] = query.Terms
	}
	if filter := strings.TrimSpace(query.Filters["filter"]); filter != "" {
		params["filter"] = filter
	}
	if strings.TrimSpace(query.PageCursor) != "" {
		params["cursor"] = strings.TrimSpace(query.PageCursor)
	}
	body, err := c.http.Get(ctx, "/works", params)
	if err != nil {
		return SourceResponse{}, err
	}
	var payload openAlexWorksResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := make([]SourceRecord, 0, len(payload.Results))
	for _, work := range payload.Results {
		openAlexID := normalizeOpenAlexID(work.ID)
		records = append(records, SourceRecord{
			Source:   "openalex",
			SourceID: openAlexID,
			Title:    strings.TrimSpace(work.Title),
			Identifiers: Identifiers{
				DOI:        normalizeSourceDOI(work.DOI),
				OpenAlexID: openAlexID,
			},
			Year:       work.PublicationYear,
			URLs:       nonEmptyStrings(work.PrimaryLocation.LandingPageURL),
			License:    strings.TrimSpace(work.PrimaryLocation.License),
			OpenAccess: work.OpenAccess.IsOA,
			Metadata:   openAlexWorkMetadata(work),
		})
	}
	return SourceResponse{Records: records, RawRef: rawOpenAlexRef(params), NextPageCursor: strings.TrimSpace(payload.Meta.NextCursor)}, nil
}

// SearchAuthors queries OpenAlex authors.
func (c OpenAlexConnector) SearchAuthors(ctx context.Context, query SourceQuery) ([]OpenAlexEntity, string, error) {
	return c.searchEntities(ctx, "/authors", "openalex-author", query)
}

// SearchInstitutions queries OpenAlex institutions.
func (c OpenAlexConnector) SearchInstitutions(ctx context.Context, query SourceQuery) ([]OpenAlexEntity, string, error) {
	return c.searchEntities(ctx, "/institutions", "openalex-institution", query)
}

// SearchConcepts queries OpenAlex concepts.
func (c OpenAlexConnector) SearchConcepts(ctx context.Context, query SourceQuery) ([]OpenAlexEntity, string, error) {
	return c.searchEntities(ctx, "/concepts", "openalex-concept", query)
}

func (c OpenAlexConnector) searchEntities(ctx context.Context, path, source string, query SourceQuery) ([]OpenAlexEntity, string, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	params := map[string]string{"per-page": strconv.Itoa(limit)}
	if strings.TrimSpace(query.Terms) != "" {
		params["search"] = query.Terms
	}
	body, err := c.http.Get(ctx, path, params)
	if err != nil {
		return nil, "", err
	}
	var payload openAlexEntitiesResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "", err
	}
	entities := make([]OpenAlexEntity, 0, len(payload.Results))
	for _, result := range payload.Results {
		id := normalizeOpenAlexID(result.ID)
		entities = append(entities, OpenAlexEntity{Source: source, SourceID: id, DisplayName: compactSpace(result.DisplayName), WorksCount: result.WorksCount, Metadata: map[string]string{"ror": strings.TrimSpace(result.ROR), "country_code": strings.TrimSpace(result.CountryCode)}})
	}
	return entities, rawOpenAlexEntityRef(path, params), nil
}

type openAlexWorksResponse struct {
	Meta    openAlexMeta   `json:"meta"`
	Results []openAlexWork `json:"results"`
}

type openAlexEntitiesResponse struct {
	Results []openAlexEntityPayload `json:"results"`
}

type openAlexEntityPayload struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	WorksCount  int    `json:"works_count"`
	ROR         string `json:"ror"`
	CountryCode string `json:"country_code"`
}

type openAlexMeta struct {
	NextCursor string `json:"next_cursor"`
}

type openAlexWork struct {
	ID              string                  `json:"id"`
	DOI             string                  `json:"doi"`
	Title           string                  `json:"title"`
	PublicationYear int                     `json:"publication_year"`
	Type            string                  `json:"type"`
	OpenAccess      openAlexOpenAccess      `json:"open_access"`
	PrimaryLocation openAlexPrimaryLocation `json:"primary_location"`
	Concepts        []openAlexConcept       `json:"concepts"`
	RelatedWorks    []string                `json:"related_works"`
	ReferencedWorks []string                `json:"referenced_works"`
	PrimaryTopic    openAlexTopic           `json:"primary_topic"`
}

type openAlexOpenAccess struct {
	IsOA     bool   `json:"is_oa"`
	OAStatus string `json:"oa_status"`
}

type openAlexPrimaryLocation struct {
	LandingPageURL string `json:"landing_page_url"`
	License        string `json:"license"`
}

type openAlexConcept struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	Score       float64 `json:"score"`
}

type openAlexTopic struct {
	DisplayName string              `json:"display_name"`
	Domain      openAlexTopicBucket `json:"domain"`
	Field       openAlexTopicBucket `json:"field"`
	Subfield    openAlexTopicBucket `json:"subfield"`
}

type openAlexTopicBucket struct {
	DisplayName string `json:"display_name"`
}

func openAlexWorkMetadata(work openAlexWork) map[string]string {
	metadata := map[string]string{
		"type":                 strings.TrimSpace(work.Type),
		"oa_status":            strings.TrimSpace(work.OpenAccess.OAStatus),
		"concepts":             strings.Join(openAlexConcepts(work.Concepts), "; "),
		"concept_ids":          strings.Join(openAlexConceptIDs(work.Concepts), "; "),
		"related_openalex_ids": strings.Join(normalizeOpenAlexIDs(work.RelatedWorks), "; "),
	}
	if len(work.Concepts) > 0 {
		metadata["top_concept"] = compactSpace(work.Concepts[0].DisplayName)
	}
	if topic := compactSpace(work.PrimaryTopic.DisplayName); topic != "" {
		metadata["topic"] = topic
	}
	if domain := compactSpace(work.PrimaryTopic.Domain.DisplayName); domain != "" {
		metadata["domain"] = domain
	}
	if field := compactSpace(work.PrimaryTopic.Field.DisplayName); field != "" {
		metadata["field"] = field
	}
	if subfield := compactSpace(work.PrimaryTopic.Subfield.DisplayName); subfield != "" {
		metadata["subfield"] = subfield
	}
	return metadata
}

func openAlexConcepts(concepts []openAlexConcept) []string {
	out := []string{}
	for _, concept := range concepts {
		name := compactSpace(concept.DisplayName)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

func openAlexConceptIDs(concepts []openAlexConcept) []string {
	out := []string{}
	for _, concept := range concepts {
		id := normalizeOpenAlexID(concept.ID)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

// RelatedWorks returns related OpenAlex work IDs as lightweight discovery records.
func (c OpenAlexConnector) RelatedWorks(ctx context.Context, workID string, limit int) (SourceResponse, error) {
	workID = normalizeOpenAlexID(workID)
	if workID == "" {
		return SourceResponse{}, fmt.Errorf("openalex work id is required")
	}
	if limit <= 0 {
		limit = 25
	}
	work, err := c.fetchWork(ctx, workID)
	if err != nil {
		return SourceResponse{}, err
	}
	records := []SourceRecord{}
	for _, relatedID := range normalizeOpenAlexIDs(work.RelatedWorks) {
		records = append(records, SourceRecord{Source: "openalex", SourceID: relatedID, Title: relatedID, Identifiers: Identifiers{OpenAlexID: relatedID}, Metadata: map[string]string{"discovery": "related_work", "related_to": workID}})
		if len(records) >= limit {
			break
		}
	}
	return SourceResponse{Records: records, RawRef: fmt.Sprintf("openalex:/works/%s/related?limit=%d", workID, limit)}, nil
}

// OpenAlexGraphQuery describes an OpenAlex citation graph expansion.
type OpenAlexGraphQuery struct {
	WorkID    string
	Limit     int
	Direction SemanticScholarGraphDirection
}

// ExpandCitationGraph fetches OpenAlex references, citations, or both for one work.
func (c OpenAlexConnector) ExpandCitationGraph(ctx context.Context, query OpenAlexGraphQuery) (CitationGraphExpansion, error) {
	workID := normalizeOpenAlexID(query.WorkID)
	if workID == "" {
		return CitationGraphExpansion{}, fmt.Errorf("openalex work id is required")
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	direction := query.Direction
	if direction == "" {
		direction = SemanticScholarDirectionBoth
	}
	expansion := CitationGraphExpansion{SeedID: workID, Records: map[string]SourceRecord{}, RawRef: rawOpenAlexGraphRef(workID, direction, limit)}
	if direction == SemanticScholarDirectionReferences || direction == SemanticScholarDirectionBoth {
		work, err := c.fetchWork(ctx, workID)
		if err != nil {
			return CitationGraphExpansion{}, err
		}
		for _, ref := range normalizeOpenAlexIDs(work.ReferencedWorks) {
			expansion.Edges = append(expansion.Edges, CitationEdge{SourceID: workID, TargetID: ref})
			expansion.Records[ref] = SourceRecord{Source: "openalex", SourceID: ref, Title: ref, Identifiers: Identifiers{OpenAlexID: ref}, Metadata: map[string]string{"graph_role": "reference"}}
			if len(expansion.Records) >= limit {
				break
			}
		}
	}
	if direction == SemanticScholarDirectionCitations || direction == SemanticScholarDirectionBoth {
		response, err := c.Search(ctx, SourceQuery{Terms: "", Limit: limit, Filters: map[string]string{"filter": "cites:" + workID}})
		if err != nil {
			return CitationGraphExpansion{}, err
		}
		for _, record := range response.Records {
			if record.SourceID == "" {
				continue
			}
			expansion.Edges = append(expansion.Edges, CitationEdge{SourceID: record.SourceID, TargetID: workID})
			expansion.Records[record.SourceID] = record
		}
	}
	return expansion, nil
}

func (c OpenAlexConnector) fetchWork(ctx context.Context, workID string) (openAlexWork, error) {
	body, err := c.http.Get(ctx, "/works/"+url.PathEscape(workID), map[string]string{})
	if err != nil {
		return openAlexWork{}, err
	}
	var work openAlexWork
	if err := json.Unmarshal(body, &work); err != nil {
		return openAlexWork{}, err
	}
	return work, nil
}

func rawOpenAlexGraphRef(workID string, direction SemanticScholarGraphDirection, limit int) string {
	return fmt.Sprintf("openalex:/works/%s/%s?limit=%d", workID, direction, limit)
}

func rawOpenAlexRef(params map[string]string) string {
	return rawOpenAlexEntityRef("/works", params)
}

func rawOpenAlexEntityRef(path string, params map[string]string) string {
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return fmt.Sprintf("openalex:%s?%s", path, values.Encode())
}

func nonEmptyStrings(values ...string) []string {
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func normalizeOpenAlexIDs(values []string) []string {
	out := []string{}
	for _, value := range values {
		if id := normalizeOpenAlexID(value); id != "" {
			out = append(out, id)
		}
	}
	return out
}

func normalizeOpenAlexID(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "https://openalex.org/")
	return value
}

func normalizeSourceDOI(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "https://doi.org/")
	value = strings.TrimPrefix(value, "http://doi.org/")
	value = strings.TrimPrefix(value, "doi:")
	return strings.TrimSpace(value)
}
