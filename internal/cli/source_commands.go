package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/citations"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

func executeCitations(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge citations <expand|report>")
	}
	if args[0] == "report" {
		graphPath, outPath, ok := parseCitationsReport(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge citations report --graph <graph.json> --out <report.md>")
		}
		data, err := os.ReadFile(graphPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_report_read_failed", err.Error())
		}
		report, err := citations.BuildGraphReport(data)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_report_failed", err.Error())
		}
		markdown := citations.GraphReportMarkdown(report)
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_report_write_failed", err.Error())
		}
		if err := os.WriteFile(outPath, []byte(markdown), 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_report_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"path": outPath, "report": report})
		}
		fmt.Fprintf(stdout, "wrote citation graph report to %s\n", outPath)
		return 0
	}
	if args[0] != "expand" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge citations <expand|report>")
	}
	source, paperID, direction, out, limit, depth, maxRecords, importLibrary, ok := parseCitationsExpand(args[1:])
	if !ok || (source != "semantic-scholar" && source != "openalex") {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge citations expand --source semantic-scholar|openalex --paper <id> --direction <references|citations|both> --out <file>")
	}
	var expansion sources.CitationGraphExpansion
	var err error
	if source == "openalex" {
		baseURL := os.Getenv("RFORGE_OPENALEX_URL")
		if baseURL == "" {
			baseURL = "https://api.openalex.org"
		}
		expansion, err = sources.NewOpenAlexConnector(defaultSourceHTTPClient(baseURL)).ExpandCitationGraph(context.Background(), sources.OpenAlexGraphQuery{WorkID: paperID, Direction: sources.SemanticScholarGraphDirection(direction), Limit: limit})
	} else {
		connector := sources.NewSemanticScholarConnector(defaultSemanticScholarHTTPClient())
		expansion, err = expandSemanticScholarRecursive(context.Background(), connector, paperID, sources.SemanticScholarGraphDirection(direction), limit, depth, maxRecords)
	}
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
	if opts.Project != "" {
		now := time.Now().UTC()
		if err := provenance.Append(opts.Project, provenance.Event{
			SchemaVersion: "1",
			ID:            "evt_" + now.Format("20060102T150405Z") + "_citations_expand",
			Timestamp:     now.Format(time.RFC3339),
			Actor:         "rforge",
			Action:        "citations.expand",
			Target:        paperID,
			Inputs: map[string]any{
				"source":     source,
				"paper":      paperID,
				"direction":  direction,
				"limit":      limit,
				"depth":      depth,
				"maxRecords": maxRecords,
			},
			Outputs: map[string]any{
				"path":     out,
				"edges":    len(expansion.Edges),
				"records":  len(expansion.Records),
				"imported": imported,
				"rawRef":   expansion.RawRef,
			},
			Warnings: []string{},
		}); err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_provenance_failed", err.Error())
		}
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"path": out, "edges": len(expansion.Edges), "rawRef": expansion.RawRef, "imported": imported, "depth": depth, "maxRecords": maxRecords})
	}
	fmt.Fprintf(stdout, "wrote citation graph with %d edges to %s\n", len(expansion.Edges), out)
	return 0
}

func expandSemanticScholarRecursive(ctx context.Context, connector sources.SemanticScholarConnector, seedID string, direction sources.SemanticScholarGraphDirection, limit, depth, maxRecords int) (sources.CitationGraphExpansion, error) {
	if depth <= 0 {
		depth = 1
	}
	aggregate := sources.CitationGraphExpansion{SeedID: seedID, Records: map[string]sources.SourceRecord{}, RawRef: fmt.Sprintf("semantic-scholar:/recursive?seed=%s&direction=%s&limit=%d&depth=%d&max_records=%d", seedID, direction, limit, depth, maxRecords)}
	visited := map[string]bool{}
	seenEdges := map[string]bool{}
	frontier := []string{seedID}
	for level := 0; level < depth && len(frontier) > 0; level++ {
		nextSet := map[string]bool{}
		for _, paperID := range frontier {
			if visited[paperID] {
				continue
			}
			visited[paperID] = true
			expansion, err := connector.ExpandCitationGraph(ctx, sources.SemanticScholarGraphQuery{PaperID: paperID, Direction: direction, Limit: limit})
			if err != nil {
				return sources.CitationGraphExpansion{}, err
			}
			for _, edge := range expansion.Edges {
				neighbors := recursiveNeighbors(edge, paperID, direction)
				if maxRecords > 0 && !canIncludeEdgeRecords(aggregate.Records, expansion.Records, edge, maxRecords) {
					continue
				}
				for _, id := range []string{edge.SourceID, edge.TargetID} {
					if record, ok := expansion.Records[id]; ok {
						aggregate.Records[id] = record
					}
				}
				key := edge.SourceID + "\x00" + edge.TargetID
				if !seenEdges[key] {
					seenEdges[key] = true
					aggregate.Edges = append(aggregate.Edges, edge)
				}
				for _, neighbor := range neighbors {
					if !visited[neighbor] {
						nextSet[neighbor] = true
					}
				}
			}
		}
		frontier = sortedStringSet(nextSet)
	}
	return aggregate, nil
}

func canIncludeEdgeRecords(existing map[string]sources.SourceRecord, candidates map[string]sources.SourceRecord, edge sources.CitationEdge, maxRecords int) bool {
	needed := 0
	for _, id := range []string{edge.SourceID, edge.TargetID} {
		if _, already := existing[id]; already {
			continue
		}
		if _, candidate := candidates[id]; candidate {
			needed++
		}
	}
	return len(existing)+needed <= maxRecords
}

func recursiveNeighbors(edge sources.CitationEdge, paperID string, direction sources.SemanticScholarGraphDirection) []string {
	switch direction {
	case sources.SemanticScholarDirectionReferences:
		if edge.SourceID == paperID {
			return []string{edge.TargetID}
		}
	case sources.SemanticScholarDirectionCitations:
		if edge.TargetID == paperID {
			return []string{edge.SourceID}
		}
	case sources.SemanticScholarDirectionBoth:
		if edge.SourceID == paperID {
			return []string{edge.TargetID}
		}
		if edge.TargetID == paperID {
			return []string{edge.SourceID}
		}
	}
	return nil
}

func sortedStringSet(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
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
	if len(args) > 0 && args[0] == "import" {
		return executeSearchImport(args[1:], stdout, stderr, opts)
	}
	if len(args) > 0 && args[0] == "related" {
		return executeSearchRelated(args[1:], stdout, stderr, opts)
	}
	source, query, limit, filters, ok := parseSearch(args)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge search --source openalex --query <query> [--limit N] [--category arxiv-category] [--filter source-filter] [--entity authors|institutions]")
	}
	if source == "openalex" && filters["entity"] != "" {
		return executeOpenAlexEntitySearch(query, limit, filters["entity"], stdout, stderr, opts)
	}
	connector, ok := searchConnector(source)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "unknown_source", fmt.Sprintf("unknown source %q", source))
	}
	response, err := connector.Search(context.Background(), sources.SourceQuery{Terms: query, Limit: limit, Filters: filters})
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

func executeOpenAlexEntitySearch(query string, limit int, entity string, stdout, stderr io.Writer, opts globalOptions) int {
	baseURL := os.Getenv("RFORGE_OPENALEX_URL")
	if baseURL == "" {
		baseURL = "https://api.openalex.org"
	}
	connector := sources.NewOpenAlexConnector(defaultSourceHTTPClient(baseURL))
	var entities []sources.OpenAlexEntity
	var rawRef string
	var err error
	switch entity {
	case "authors":
		entities, rawRef, err = connector.SearchAuthors(context.Background(), sources.SourceQuery{Terms: query, Limit: limit})
	case "institutions":
		entities, rawRef, err = connector.SearchInstitutions(context.Background(), sources.SourceQuery{Terms: query, Limit: limit})
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge search --source openalex --query <query> --entity authors|institutions")
	}
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "search_failed", fmt.Sprintf("search: %v", err))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"entities": entities, "source": "openalex", "entity": entity, "rawRef": rawRef})
	}
	for _, entity := range entities {
		fmt.Fprintf(stdout, "%s\t%s\t%d\n", entity.SourceID, entity.DisplayName, entity.WorksCount)
	}
	return 0
}

func executeSearchRelated(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	source, paperID, limit, ok := parseSearchRelated(args)
	if !ok || source != "openalex" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge search related --source openalex --paper <work-id> [--limit N]")
	}
	baseURL := os.Getenv("RFORGE_OPENALEX_URL")
	if baseURL == "" {
		baseURL = "https://api.openalex.org"
	}
	response, err := sources.NewOpenAlexConnector(defaultSourceHTTPClient(baseURL)).RelatedWorks(context.Background(), paperID, limit)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "search_related_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"records": response.Records, "source": source, "rawRef": response.RawRef})
	}
	for _, record := range response.Records {
		fmt.Fprintf(stdout, "%s\t%s\n", record.SourceID, record.Title)
	}
	return 0
}

func executeSearchImport(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for search import")
	}
	source, query, pages, limit, filters, ok := parseSearchImport(args)
	if !ok || source != "openalex" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> search import --source openalex --query <query> --pages N [--limit N] [--filter source-filter]")
	}
	connector, ok := searchConnector(source)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "unknown_source", fmt.Sprintf("unknown source %q", source))
	}
	store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_open_failed", err.Error())
	}
	cursor := "*"
	imported, skippedDuplicate, skippedNoIdentifier := 0, 0, 0
	rawRefs := []string{}
	for page := 0; page < pages; page++ {
		response, err := connector.Search(context.Background(), sources.SourceQuery{Terms: query, Limit: limit, PageCursor: cursor, Filters: filters})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "search_import_failed", err.Error())
		}
		rawRefs = append(rawRefs, response.RawRef)
		papers, err := sources.PaperRecords(response)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "search_import_normalize_failed", err.Error())
		}
		summary, err := store.ImportRecords(papers)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "search_import_store_failed", err.Error())
		}
		imported += summary.Imported
		skippedDuplicate += len(summary.SkippedDuplicate)
		skippedNoIdentifier += summary.SkippedNoIdentifier
		if strings.TrimSpace(response.NextPageCursor) == "" {
			break
		}
		cursor = response.NextPageCursor
	}
	if err := recordDuplicateEvent(opts.Project, "search.import", map[string]any{"source": source, "query": query, "pages": pages, "limit": limit, "filters": filters}, map[string]any{"imported": imported, "skippedDuplicate": skippedDuplicate, "skippedNoIdentifier": skippedNoIdentifier, "rawRefs": rawRefs}); err != nil {
		return writeError(stdout, stderr, opts, 1, "search_import_provenance_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"source": source, "imported": imported, "skippedDuplicate": skippedDuplicate, "skippedNoIdentifier": skippedNoIdentifier, "rawRefs": rawRefs})
	}
	fmt.Fprintf(stdout, "imported %d records from %s\n", imported, source)
	return 0
}

type sourceConnector interface {
	Name() string
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
	case "europepmc":
		baseURL := os.Getenv("RFORGE_EUROPEPMC_URL")
		if baseURL == "" {
			baseURL = "https://www.ebi.ac.uk/europepmc"
		}
		return sources.NewEuropePMCConnector(defaultSourceHTTPClient(baseURL)), true
	case "pubmed":
		baseURL := os.Getenv("RFORGE_PUBMED_URL")
		if baseURL == "" {
			baseURL = "https://eutils.ncbi.nlm.nih.gov"
		}
		return sources.NewPubMedConnectorWithOptions(defaultSourceHTTPClient(baseURL), sources.PubMedOptions{
			APIKey: os.Getenv("RFORGE_PUBMED_API_KEY"),
			Tool:   os.Getenv("RFORGE_PUBMED_TOOL"),
			Email:  os.Getenv("RFORGE_PUBMED_EMAIL"),
		}), true
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
	options := sources.HTTPClientOptions{
		BaseURL:    baseURL,
		UserAgent:  "ResearchForge/dev",
		Timeout:    10 * time.Second,
		MaxRetries: envInt("RFORGE_SEMANTIC_SCHOLAR_MAX_RETRIES", 2),
	}
	if apiKey := strings.TrimSpace(os.Getenv("RFORGE_SEMANTIC_SCHOLAR_API_KEY")); apiKey != "" {
		options.Headers = map[string]string{"x-api-key": apiKey}
	}
	return sources.NewHTTPClient(options)
}

func envInt(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func parseCitationsReport(args []string) (string, string, bool) {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--graph", "--out":
			if i+1 >= len(args) {
				return "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", false
		}
	}
	return values["--graph"], values["--out"], values["--graph"] != "" && values["--out"] != ""
}

func parseCitationsExpand(args []string) (string, string, string, string, int, int, int, bool, bool) {
	values := map[string]string{}
	limit := 25
	depth := 1
	maxRecords := 0
	importLibrary := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--import-library":
			importLibrary = true
		case "--source", "--paper", "--direction", "--out", "--limit", "--depth", "--max-records":
			if i+1 >= len(args) {
				return "", "", "", "", 0, 0, 0, false, false
			}
			if args[i] == "--limit" || args[i] == "--depth" || args[i] == "--max-records" {
				parsed, err := strconv.Atoi(args[i+1])
				if err != nil || parsed <= 0 {
					return "", "", "", "", 0, 0, 0, false, false
				}
				if args[i] == "--limit" {
					limit = parsed
				} else if args[i] == "--depth" {
					depth = parsed
				} else {
					maxRecords = parsed
				}
			} else {
				values[args[i]] = args[i+1]
			}
			i++
		default:
			return "", "", "", "", 0, 0, 0, false, false
		}
	}
	direction := values["--direction"]
	if direction == "" {
		direction = "both"
	}
	validDirection := direction == string(sources.SemanticScholarDirectionReferences) || direction == string(sources.SemanticScholarDirectionCitations) || direction == string(sources.SemanticScholarDirectionBoth)
	return values["--source"], values["--paper"], direction, values["--out"], limit, depth, maxRecords, importLibrary, values["--source"] != "" && values["--paper"] != "" && values["--out"] != "" && validDirection
}

func parseSearchRelated(args []string) (string, string, int, bool) {
	limit := 25
	var source, paper string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--source":
			if i+1 >= len(args) {
				return "", "", 0, false
			}
			source = args[i+1]
			i++
		case "--paper":
			if i+1 >= len(args) {
				return "", "", 0, false
			}
			paper = args[i+1]
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
	return source, paper, limit, source != "" && strings.TrimSpace(paper) != ""
}

func parseSearchImport(args []string) (string, string, int, int, map[string]string, bool) {
	limit := 25
	pages := 1
	filters := map[string]string{}
	var source, query string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--source":
			if i+1 >= len(args) {
				return "", "", 0, 0, nil, false
			}
			source = args[i+1]
			i++
		case "--query":
			if i+1 >= len(args) {
				return "", "", 0, 0, nil, false
			}
			query = args[i+1]
			i++
		case "--pages":
			if i+1 >= len(args) {
				return "", "", 0, 0, nil, false
			}
			parsed, err := strconv.Atoi(args[i+1])
			if err != nil || parsed <= 0 {
				return "", "", 0, 0, nil, false
			}
			pages = parsed
			i++
		case "--limit":
			if i+1 >= len(args) {
				return "", "", 0, 0, nil, false
			}
			parsed, err := strconv.Atoi(args[i+1])
			if err != nil || parsed <= 0 {
				return "", "", 0, 0, nil, false
			}
			limit = parsed
			i++
		case "--filter":
			if i+1 >= len(args) {
				return "", "", 0, 0, nil, false
			}
			filters["filter"] = appendCommaFilter(filters["filter"], args[i+1])
			i++
		default:
			return "", "", 0, 0, nil, false
		}
	}
	return source, query, pages, limit, filters, source != "" && strings.TrimSpace(query) != ""
}

func parseSearch(args []string) (string, string, int, map[string]string, bool) {
	limit := 25
	filters := map[string]string{}
	var source, query string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--source":
			if i+1 >= len(args) {
				return "", "", 0, nil, false
			}
			source = args[i+1]
			i++
		case "--query":
			if i+1 >= len(args) {
				return "", "", 0, nil, false
			}
			query = args[i+1]
			i++
		case "--limit":
			if i+1 >= len(args) {
				return "", "", 0, nil, false
			}
			parsed, err := strconv.Atoi(args[i+1])
			if err != nil || parsed <= 0 {
				return "", "", 0, nil, false
			}
			limit = parsed
			i++
		case "--category":
			if i+1 >= len(args) {
				return "", "", 0, nil, false
			}
			filters["category"] = args[i+1]
			i++
		case "--filter":
			if i+1 >= len(args) {
				return "", "", 0, nil, false
			}
			filters["filter"] = appendCommaFilter(filters["filter"], args[i+1])
			i++
		case "--entity":
			if i+1 >= len(args) {
				return "", "", 0, nil, false
			}
			filters["entity"] = args[i+1]
			i++
		default:
			return "", "", 0, nil, false
		}
	}
	return source, query, limit, filters, source != "" && (strings.TrimSpace(query) != "" || strings.TrimSpace(filters["category"]) != "" || strings.TrimSpace(filters["filter"]) != "")
}

func appendCommaFilter(existing, next string) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return strings.TrimSpace(existing)
	}
	if strings.TrimSpace(existing) == "" {
		return next
	}
	return strings.TrimSpace(existing) + "," + next
}
