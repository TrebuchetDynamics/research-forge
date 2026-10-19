package sources

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

// PaperResolution is the merged result of resolving one paper across sources.
type PaperResolution struct {
	Input    string              `json:"input"`
	Paper    library.PaperRecord `json:"paper"`
	Sources  []ResolutionSource  `json:"sources"`
	OA       *OAResult           `json:"oa,omitempty"`
	Resolved bool                `json:"resolved"`
}

// ResolutionSource records one source that contributed metadata.
type ResolutionSource struct {
	Source  string `json:"source"`
	Title   string `json:"title"`
	RawRef  string `json:"rawRef"`
	MatchBy string `json:"matchBy"`
}

// OAResult shows OA availability snapshot.
type OAResult struct {
	IsOA        bool     `json:"isOA"`
	License     string   `json:"license,omitempty"`
	PDFURL      string   `json:"pdfURL,omitempty"`
	OAHost      string   `json:"oaHost,omitempty"`
	OAStatus    string   `json:"oaStatus,omitempty"`
	OtherOAURLs []string `json:"otherOAURLs,omitempty"`
	Unavailable bool     `json:"unavailable,omitempty"`
}

// classifyPaperInput returns the likely identifier type of a paper input string.
func classifyPaperInput(input string) string {
	input = strings.TrimSpace(input)
	switch {
	case strings.HasPrefix(input, "10.") && strings.Contains(input, "/"):
		return "doi"
	case strings.HasPrefix(input, "W"):
		return "openalex"
	case strings.HasPrefix(input, "arXiv:") || strings.HasPrefix(input, "http://arxiv.org/") || strings.HasPrefix(input, "https://arxiv.org/"):
		return "arxiv"
	case len(input) >= 5 && len(input) <= 20 && isArXivBareID(input):
		return "arxiv"
	case strings.HasPrefix(input, "PMID:") || (len(input) >= 1 && len(input) <= 8 && onlyDigits(input)):
		return "pmid"
	case strings.HasPrefix(input, "PMC"):
		return "pmcid"
	case strings.HasPrefix(input, "CorpusID:"):
		return "semantic-scholar"
	case strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://"):
		return "url"
	default:
		return "title"
	}
}

func onlyDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

func isArXivBareID(input string) bool {
	parts := strings.Split(input, ".")
	if len(parts) < 1 || len(parts) > 2 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 {
			return false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

// ResolvePaper resolves a paper identifier across OpenAlex, Crossref, arXiv, and
// Semantic Scholar, merging metadata from all sources.
func ResolvePaper(ctx context.Context, httpClient HTTPClient, input string) (PaperResolution, error) {
	input = strings.TrimSpace(input)
	kind := classifyPaperInput(input)

	result := PaperResolution{
		Input:    input,
		Sources:  []ResolutionSource{},
		Resolved: false,
	}

	// Keep explicitly injected clients intact; otherwise route each connector to
	// its configured endpoint instead of sending requests with an empty base URL.
	clientFor := func(env, fallback string) HTTPClient {
		client := httpClient
		if client.baseURL == "" {
			client.baseURL = strings.TrimRight(strings.TrimSpace(os.Getenv(env)), "/")
			if client.baseURL == "" {
				client.baseURL = fallback
			}
		}
		return client
	}
	oa := NewOpenAlexConnector(clientFor("RFORGE_OPENALEX_URL", "https://api.openalex.org"))
	cr := NewCrossrefConnector(clientFor("RFORGE_CROSSREF_URL", "https://api.crossref.org"))
	ar := NewArXivConnector(clientFor("RFORGE_ARXIV_URL", "https://export.arxiv.org"))
	ss := NewSemanticScholarConnector(clientFor("RFORGE_SEMANTIC_SCHOLAR_URL", "https://api.semanticscholar.org"))

	type lookupResult struct {
		record  SourceRecord
		rawRef  string
		kind    string
		matchBy string
	}
	var lookups []lookupResult

	switch kind {
	case "doi":
		doi := normalizeSourceDOI(input)
		if doi == "" {
			return result, nil
		}
		// OpenAlex
		response, err := oa.Search(ctx, SourceQuery{Terms: "", Limit: 1, Filters: map[string]string{"filter": "doi:" + doi}})
		if err == nil && len(response.Records) > 0 {
			lookups = append(lookups, lookupResult{response.Records[0], response.RawRef, "openalex", "doi"})
		}
		// Crossref
		record, ref, err := cr.LookupDOI(ctx, doi)
		if err == nil {
			lookups = append(lookups, lookupResult{record, ref, "crossref", "doi"})
		}
		// Unpaywall OA check
		upw := NewUnpaywallConnector(clientFor("RFORGE_UNPAYWALL_URL", "https://api.unpaywall.org"), os.Getenv("RFORGE_UNPAYWALL_EMAIL"))
		oaRec, oaErr := upw.LookupDOI(ctx, doi)
		if oaErr == nil {
			result.OA = &OAResult{
				IsOA:     oaRec.OpenAccess,
				License:  oaRec.License,
				PDFURL:   oaRec.BestURL,
				OAHost:   oaRec.SourceRef.Source,
				OAStatus: oaRec.OAStatus,
			}
		}

	case "arxiv":
		arID := normalizeArXivID(input)
		record, ref, err := ar.LookupPaperByArXivID(ctx, arID)
		if err == nil {
			lookups = append(lookups, lookupResult{record, ref, "arxiv", "arxiv-id"})
		}
		// also try OpenAlex
		response, err := oa.Search(ctx, SourceQuery{Terms: "", Limit: 1, Filters: map[string]string{"filter": "arxiv:" + arID}})
		if err == nil && len(response.Records) > 0 {
			lookups = append(lookups, lookupResult{response.Records[0], response.RawRef, "openalex", "arxiv-id"})
		}

	case "openalex":
		response, err := oa.Search(ctx, SourceQuery{Terms: "", Limit: 1, Filters: map[string]string{"filter": "openalex:" + input}})
		if err == nil && len(response.Records) > 0 {
			lookups = append(lookups, lookupResult{response.Records[0], response.RawRef, "openalex", "openalex-id"})
			if doi := response.Records[0].Identifiers.DOI; doi != "" {
				crRecord, crRef, crErr := cr.LookupDOI(ctx, doi)
				if crErr == nil {
					lookups = append(lookups, lookupResult{crRecord, crRef, "crossref", "doi"})
				}
			}
		}

	case "url":
		extractedDOI := extractDOIFromURL(input)
		extractedArXiv := extractArXivIDFromURL(input)
		if extractedDOI != "" {
			doi := normalizeSourceDOI(extractedDOI)
			response, err := oa.Search(ctx, SourceQuery{Terms: "", Limit: 1, Filters: map[string]string{"filter": "doi:" + doi}})
			if err == nil && len(response.Records) > 0 {
				lookups = append(lookups, lookupResult{response.Records[0], response.RawRef, "openalex", "doi"})
			}
			record, ref, err := cr.LookupDOI(ctx, doi)
			if err == nil {
				lookups = append(lookups, lookupResult{record, ref, "crossref", "doi"})
			}
		} else if extractedArXiv != "" {
			record, ref, err := ar.LookupPaperByArXivID(ctx, extractedArXiv)
			if err == nil {
				lookups = append(lookups, lookupResult{record, ref, "arxiv", "arxiv-id"})
			}
		}

	default: // title
		response, err := oa.Search(ctx, SourceQuery{Terms: input, Limit: 5})
		if err == nil {
			for _, rec := range response.Records {
				if titleMatch(rec.Title, input) {
					lookups = append(lookups, lookupResult{rec, response.RawRef, "openalex", "title"})
				}
			}
		}
		response, err = ss.Search(ctx, SourceQuery{Terms: input, Limit: 5})
		if err == nil {
			for _, rec := range response.Records {
				if titleMatch(rec.Title, input) {
					lookups = append(lookups, lookupResult{rec, response.RawRef, "semantic-scholar", "title"})
				}
			}
		}
	}

	if len(lookups) == 0 {
		return result, nil
	}

	// Merge: collect identifiers across all lookups, pick best record
	best := lookups[0].record
	seenSources := map[string]bool{}
	for _, l := range lookups {
		if seenSources[l.kind] {
			// deduplicate same source, merge identifiers
			if l.record.Identifiers.DOI != "" {
				best.Identifiers.DOI = l.record.Identifiers.DOI
			}
			if l.record.Identifiers.OpenAlexID != "" {
				best.Identifiers.OpenAlexID = l.record.Identifiers.OpenAlexID
			}
			if l.record.Identifiers.ArXivID != "" {
				best.Identifiers.ArXivID = l.record.Identifiers.ArXivID
			}
			if l.record.Identifiers.PMID != "" {
				best.Identifiers.PMID = l.record.Identifiers.PMID
			}
			if best.Abstract == "" && l.record.Abstract != "" {
				best.Abstract = l.record.Abstract
			}
			if best.Venue == "" && l.record.Venue != "" {
				best.Venue = l.record.Venue
			}
			if best.Publisher == "" && l.record.Publisher != "" {
				best.Publisher = l.record.Publisher
			}
			continue
		}
		seenSources[l.kind] = true
		result.Sources = append(result.Sources, ResolutionSource{
			Source:  l.kind,
			Title:   l.record.Title,
			RawRef:  l.rawRef,
			MatchBy: l.matchBy,
		})
	}

	paper, err := library.NewPaperRecord(library.PaperRecordInput{
		Title:    best.Title,
		Abstract: best.Abstract,
		Identifiers: library.Identifiers{
			DOI:               best.Identifiers.DOI,
			OpenAlexID:        best.Identifiers.OpenAlexID,
			ArXivID:           best.Identifiers.ArXivID,
			PMID:              best.Identifiers.PMID,
			PMCID:             best.Identifiers.PMCID,
			CrossrefID:        best.Identifiers.CrossrefID,
			SemanticScholarID: best.Identifiers.SemanticScholarID,
		},
		Year:       best.Year,
		Venue:      best.Venue,
		Publisher:  best.Publisher,
		URLs:       best.URLs,
		License:    best.License,
		OpenAccess: best.OpenAccess,
	})
	if err != nil {
		return result, fmt.Errorf("merge paper record: %w", err)
	}

	sort.Slice(result.Sources, func(i, j int) bool {
		return result.Sources[i].Source < result.Sources[j].Source
	})

	result.Paper = paper
	result.Resolved = true

	// If no OA from Unpaywall, check OA status from merged metadata
	if result.OA == nil && best.OpenAccess {
		result.OA = &OAResult{
			IsOA:    true,
			License: best.License,
		}
	}

	return result, nil
}

// titleMatch reports whether a search result title approximates the query.
func titleMatch(resultTitle, query string) bool {
	rt := strings.ToLower(strings.TrimSpace(resultTitle))
	q := strings.ToLower(strings.TrimSpace(query))
	return strings.Contains(rt, q) || strings.Contains(q, rt)
}

func extractDOIFromURL(rawURL string) string {
	if idx := strings.Index(rawURL, "doi.org/"); idx >= 0 {
		doi := rawURL[idx+8:]
		if idx2 := strings.IndexAny(doi, " ?#"); idx2 >= 0 {
			doi = doi[:idx2]
		}
		return normalizeSourceDOI(doi)
	}
	return ""
}

func extractArXivIDFromURL(rawURL string) string {
	markers := []string{"arxiv.org/abs/", "arxiv.org/pdf/", "arxiv.org/"}
	for _, m := range markers {
		if idx := strings.Index(rawURL, m); idx >= 0 {
			id := rawURL[idx+len(m):]
			if idx2 := strings.IndexAny(id, " ?#"); idx2 >= 0 {
				id = id[:idx2]
			}
			if norm := normalizeArXivID(id); norm != "" {
				return norm
			}
		}
	}
	return ""
}
