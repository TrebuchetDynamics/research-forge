package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

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
