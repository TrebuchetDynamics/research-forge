package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/citations"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

func executeCitations(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 || args[0] != "expand" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge citations expand --source semantic-scholar --paper <id> --direction <references|citations|both> --out <file>")
	}
	source, paperID, direction, out, limit, importLibrary, ok := parseCitationsExpand(args[1:])
	if !ok || source != "semantic-scholar" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge citations expand --source semantic-scholar --paper <id> --direction <references|citations|both> --out <file>")
	}
	connector := sources.NewSemanticScholarConnector(defaultSemanticScholarHTTPClient())
	expansion, err := connector.ExpandCitationGraph(context.Background(), sources.SemanticScholarGraphQuery{PaperID: paperID, Direction: sources.SemanticScholarGraphDirection(direction), Limit: limit})
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "citation_expand_failed", fmt.Sprintf("expand citations: %v", err))
	}
	graph := citations.NewGraph()
	for _, edge := range expansion.Edges {
		graph.AddCitation(edge.SourceID, edge.TargetID)
	}
	data, err := graph.ExportJSON()
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "citation_graph_export_failed", err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return writeError(stdout, stderr, opts, 1, "citation_graph_write_failed", err.Error())
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return writeError(stdout, stderr, opts, 1, "citation_graph_write_failed", err.Error())
	}
	imported := 0
	if importLibrary {
		if opts.Project == "" {
			return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required when using --import-library")
		}
		store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_open_failed", err.Error())
		}
		records := make([]sources.SourceRecord, 0, len(expansion.Records))
		for _, record := range expansion.Records {
			records = append(records, record)
		}
		papers, err := sources.PaperRecords(sources.SourceResponse{Records: records, RawRef: expansion.RawRef})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_import_normalize_failed", err.Error())
		}
		summary, err := store.ImportRecords(papers)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_import_failed", err.Error())
		}
		imported = summary.Imported
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"path": out, "edges": len(expansion.Edges), "rawRef": expansion.RawRef, "imported": imported})
	}
	fmt.Fprintf(stdout, "wrote citation graph with %d edges to %s\n", len(expansion.Edges), out)
	return 0
}

func executeOA(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 2 || args[0] != "lookup" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge oa lookup <doi>")
	}
	email := os.Getenv("RFORGE_UNPAYWALL_EMAIL")
	baseURL := os.Getenv("RFORGE_UNPAYWALL_URL")
	if baseURL == "" {
		baseURL = "https://api.unpaywall.org"
	}
	connector := sources.NewUnpaywallConnector(defaultSourceHTTPClient(baseURL), email)
	record, err := connector.LookupDOI(context.Background(), args[1])
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "oa_lookup_failed", fmt.Sprintf("open access lookup failed: %v", err))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"open_access": record})
	}
	fmt.Fprintf(stdout, "%s\t%t\t%s\t%s\n", record.DOI, record.OpenAccess, record.OAStatus, record.PDFURL)
	return 0
}

func executeSearch(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	source, query, limit, ok := parseSearch(args)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge search --source openalex --query <query> [--limit N]")
	}
	connector, ok := searchConnector(source)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "unknown_source", fmt.Sprintf("unknown source %q", source))
	}
	response, err := connector.Search(context.Background(), sources.SourceQuery{Terms: query, Limit: limit})
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "search_failed", fmt.Sprintf("search: %v", err))
	}
	papers, err := sources.PaperRecords(response)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "search_normalize_failed", fmt.Sprintf("normalize search results: %v", err))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"papers": papers, "source": source})
	}
	for _, paper := range papers {
		fmt.Fprintf(stdout, "%s\t%s\n", paper.Identifiers.DOI, paper.Title)
	}
	return 0
}

type sourceConnector interface {
	Search(context.Context, sources.SourceQuery) (sources.SourceResponse, error)
}

func searchConnector(source string) (sourceConnector, bool) {
	switch source {
	case "openalex":
		baseURL := os.Getenv("RFORGE_OPENALEX_URL")
		if baseURL == "" {
			baseURL = "https://api.openalex.org"
		}
		return sources.NewOpenAlexConnector(defaultSourceHTTPClient(baseURL)), true
	case "arxiv":
		baseURL := os.Getenv("RFORGE_ARXIV_URL")
		if baseURL == "" {
			baseURL = "https://export.arxiv.org"
		}
		return sources.NewArXivConnector(defaultSourceHTTPClient(baseURL)), true
	case "crossref":
		baseURL := os.Getenv("RFORGE_CROSSREF_URL")
		if baseURL == "" {
			baseURL = "https://api.crossref.org"
		}
		return sources.NewCrossrefConnector(defaultSourceHTTPClient(baseURL)), true
	case "semantic-scholar":
		return sources.NewSemanticScholarConnector(defaultSemanticScholarHTTPClient()), true
	default:
		return nil, false
	}
}

func defaultSourceHTTPClient(baseURL string) sources.HTTPClient {
	return sources.NewHTTPClient(sources.HTTPClientOptions{
		BaseURL:    baseURL,
		UserAgent:  "ResearchForge/dev",
		Timeout:    10 * time.Second,
		MaxRetries: 2,
	})
}

func defaultSemanticScholarHTTPClient() sources.HTTPClient {
	baseURL := os.Getenv("RFORGE_SEMANTIC_SCHOLAR_URL")
	if baseURL == "" {
		baseURL = "https://api.semanticscholar.org"
	}
	if apiKey := strings.TrimSpace(os.Getenv("RFORGE_SEMANTIC_SCHOLAR_API_KEY")); apiKey != "" {
		return sources.NewHTTPClient(sources.HTTPClientOptions{
			BaseURL:    baseURL,
			UserAgent:  "ResearchForge/dev",
			Timeout:    10 * time.Second,
			MaxRetries: 2,
			Headers:    map[string]string{"x-api-key": apiKey},
		})
	}
	return defaultSourceHTTPClient(baseURL)
}

func parseCitationsExpand(args []string) (string, string, string, string, int, bool, bool) {
	values := map[string]string{}
	limit := 25
	importLibrary := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--import-library":
			importLibrary = true
		case "--source", "--paper", "--direction", "--out", "--limit":
			if i+1 >= len(args) {
				return "", "", "", "", 0, false, false
			}
			if args[i] == "--limit" {
				parsed, err := strconv.Atoi(args[i+1])
				if err != nil || parsed <= 0 {
					return "", "", "", "", 0, false, false
				}
				limit = parsed
			} else {
				values[args[i]] = args[i+1]
			}
			i++
		default:
			return "", "", "", "", 0, false, false
		}
	}
	direction := values["--direction"]
	if direction == "" {
		direction = "both"
	}
	validDirection := direction == string(sources.SemanticScholarDirectionReferences) || direction == string(sources.SemanticScholarDirectionCitations) || direction == string(sources.SemanticScholarDirectionBoth)
	return values["--source"], values["--paper"], direction, values["--out"], limit, importLibrary, values["--source"] != "" && values["--paper"] != "" && values["--out"] != "" && validDirection
}

func parseSearch(args []string) (string, string, int, bool) {
	limit := 25
	var source, query string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--source":
			if i+1 >= len(args) {
				return "", "", 0, false
			}
			source = args[i+1]
			i++
		case "--query":
			if i+1 >= len(args) {
				return "", "", 0, false
			}
			query = args[i+1]
			i++
		case "--limit":
			if i+1 >= len(args) {
				return "", "", 0, false
			}
			parsed, err := strconv.Atoi(args[i+1])
			if err != nil || parsed <= 0 {
				return "", "", 0, false
			}
			limit = parsed
			i++
		default:
			return "", "", 0, false
		}
	}
	return source, query, limit, source != "" && strings.TrimSpace(query) != ""
}
