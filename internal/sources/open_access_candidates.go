package sources

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/research-forge/internal/library"
)

type DOAJConnector struct{ http HTTPClient }

func NewDOAJConnector(httpClient HTTPClient) DOAJConnector { return DOAJConnector{http: httpClient} }
func (DOAJConnector) Name() string                         { return "doaj" }
func (c DOAJConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	path := "/api/search/articles/" + url.PathEscape(strings.TrimSpace(query.Terms))
	body, err := c.http.Get(ctx, path, map[string]string{"pageSize": strconv.Itoa(limit)})
	if err != nil {
		return SourceResponse{}, err
	}
	var payload doajResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := []SourceRecord{}
	for _, item := range payload.Results {
		doi := ""
		for _, id := range item.BibJSON.Identifier {
			if strings.EqualFold(id.Type, "doi") {
				doi = normalizeSourceDOI(id.ID)
			}
		}
		links := []string{}
		for _, link := range item.BibJSON.Link {
			if strings.TrimSpace(link.URL) != "" {
				links = append(links, strings.TrimSpace(link.URL))
			}
		}
		license := ""
		if len(item.BibJSON.License) > 0 {
			license = strings.TrimSpace(item.BibJSON.License[0].Type)
		}
		records = append(records, SourceRecord{Source: "doaj", SourceID: strings.TrimSpace(item.ID), Title: strings.TrimSpace(item.BibJSON.Title), Identifiers: Identifiers{DOI: doi}, Year: item.BibJSON.YearInt(), Venue: item.BibJSON.Journal.Title, URLs: nonEmptyStrings(links...), License: license, OpenAccess: true, Metadata: map[string]string{"full_text_url": firstNonEmpty(links...), "license": license}})
	}
	return SourceResponse{Records: records, RawRef: "doaj:" + path}, nil
}

type COREConnector struct{ http HTTPClient }

func NewCOREConnector(httpClient HTTPClient) COREConnector { return COREConnector{http: httpClient} }
func (COREConnector) Name() string                         { return "core" }
func (c COREConnector) Search(ctx context.Context, query SourceQuery) (SourceResponse, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 25
	}
	body, err := c.http.Get(ctx, "/v3/search/works", map[string]string{"q": query.Terms, "limit": strconv.Itoa(limit)})
	if err != nil {
		return SourceResponse{}, err
	}
	var payload coreResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return SourceResponse{}, err
	}
	records := []SourceRecord{}
	for _, item := range payload.Results {
		links := []string{item.DownloadURL}
		for _, link := range item.Links {
			links = append(links, link.URL)
		}
		records = append(records, SourceRecord{Source: "core", SourceID: strings.TrimSpace(item.ID), Title: strings.TrimSpace(item.Title), Identifiers: Identifiers{DOI: normalizeSourceDOI(item.DOI)}, Year: item.YearPublished, Publisher: item.Publisher, URLs: nonEmptyStrings(links...), License: strings.TrimSpace(item.License), OpenAccess: true, Metadata: map[string]string{"download_url": strings.TrimSpace(item.DownloadURL), "license": strings.TrimSpace(item.License)}})
	}
	return SourceResponse{Records: records, RawRef: "core:/v3/search/works?q=" + url.QueryEscape(query.Terms)}, nil
}

type OpenAccessCandidateComparison struct {
	SchemaVersion string                `json:"schemaVersion"`
	Candidates    []OpenAccessCandidate `json:"candidates"`
}
type OpenAccessCandidate struct {
	PaperTitle               string `json:"paperTitle"`
	DOI                      string `json:"doi,omitempty"`
	Source                   string `json:"source"`
	URL                      string `json:"url"`
	License                  string `json:"license,omitempty"`
	OAStatus                 string `json:"oaStatus,omitempty"`
	Provenance               string `json:"provenance"`
	ReviewerApprovalRequired bool   `json:"reviewerApprovalRequired"`
}

func CompareOpenAccessCandidates(records []library.PaperRecord) OpenAccessCandidateComparison {
	out := []OpenAccessCandidate{}
	for _, record := range records {
		add := func(source, candidateURL, license, status, provenance string) {
			if strings.TrimSpace(candidateURL) != "" {
				out = append(out, OpenAccessCandidate{PaperTitle: record.Title, DOI: record.Identifiers.DOI, Source: source, URL: strings.TrimSpace(candidateURL), License: strings.TrimSpace(license), OAStatus: strings.TrimSpace(status), Provenance: provenance, ReviewerApprovalRequired: true})
			}
		}
		for _, ref := range record.SourceRefs {
			m := ref.Metadata
			switch ref.Source {
			case "unpaywall":
				add("unpaywall", firstNonEmpty(m["pdf_url"], m["url_for_pdf"], m["best_url"]), firstNonEmpty(m["license"], record.License), m["oa_status"], ref.RawPayloadRef)
			case "doaj":
				add("doaj", firstNonEmpty(m["full_text_url"], m["url"]), firstNonEmpty(m["license"], record.License), "open", ref.RawPayloadRef)
			case "core":
				add("core", firstNonEmpty(m["download_url"], m["full_text_url"], m["url"]), firstNonEmpty(m["license"], record.License), "open", ref.RawPayloadRef)
			case "europepmc", "pubmed", "pmc":
				add("pubmed-europepmc-pmc", firstNonEmpty(m["full_text_url"], firstNonEmpty(record.URLs...)), firstNonEmpty(m["license"], record.License), "open", ref.RawPayloadRef)
			}
		}
		if record.Identifiers.ArXivID != "" {
			add("arxiv", "https://arxiv.org/pdf/"+record.Identifiers.ArXivID, record.License, "preprint", "arxiv:"+record.Identifiers.ArXivID)
		}
		for _, u := range record.URLs {
			if strings.HasPrefix(u, "/") || strings.HasPrefix(u, "file:") {
				add("local", u, record.License, "local-only", "local-import")
			}
		}
	}
	return OpenAccessCandidateComparison{SchemaVersion: "1", Candidates: out}
}

type doajResponse struct {
	Results []doajItem `json:"results"`
}
type doajItem struct {
	ID      string      `json:"id"`
	BibJSON doajBibJSON `json:"bibjson"`
}
type doajBibJSON struct {
	Title   string `json:"title"`
	Year    string `json:"year"`
	Journal struct {
		Title string `json:"title"`
	} `json:"journal"`
	Identifier []struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	} `json:"identifier"`
	Link []struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"link"`
	License []struct {
		Type string `json:"type"`
	} `json:"license"`
}

func (b doajBibJSON) YearInt() int { y, _ := strconv.Atoi(strings.TrimSpace(b.Year)); return y }

type coreResponse struct {
	Results []coreItem `json:"results"`
}
type coreItem struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	DOI           string `json:"doi"`
	Publisher     string `json:"publisher"`
	DownloadURL   string `json:"downloadUrl"`
	License       string `json:"license"`
	YearPublished int    `json:"yearPublished"`
	Links         []struct {
		URL string `json:"url"`
	} `json:"links"`
}
