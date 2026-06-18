package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/research-forge/internal/citations"
	"github.com/TrebuchetDynamics/research-forge/internal/documents"
	"github.com/TrebuchetDynamics/research-forge/internal/evidence"
	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/parsing"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

func executeCitations(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge citations <expand|report|import-bibliography|domain-map>")
	}
	if args[0] == "accessible-view" {
		graphPath, domainPath, outPath, filter, format, ok := parseCitationsAccessibleView(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge citations accessible-view --graph <graph.json> --out <view.md|view.json> [--domain-map <domain-map.json> --filter <text> --format markdown|json]")
		}
		graphData, err := os.ReadFile(graphPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "accessible_graph_read_failed", err.Error())
		}
		var domain citations.DomainMapArtifact
		if domainPath != "" {
			if err := readJSONFile(domainPath, &domain); err != nil {
				return writeError(stdout, stderr, opts, 1, "accessible_domain_read_failed", err.Error())
			}
		}
		view, err := citations.BuildAccessibleGraphView(graphData, domain, citations.AccessibleGraphOptions{Filter: filter})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "accessible_graph_failed", err.Error())
		}
		if format == "json" {
			if err := writeJSONFile(outPath, view); err != nil {
				return writeError(stdout, stderr, opts, 1, "accessible_graph_write_failed", err.Error())
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return writeError(stdout, stderr, opts, 1, "accessible_graph_write_failed", err.Error())
			}
			if err := os.WriteFile(outPath, []byte(citations.AccessibleGraphMarkdown(view)), 0o644); err != nil {
				return writeError(stdout, stderr, opts, 1, "accessible_graph_write_failed", err.Error())
			}
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"accessibleGraph": view, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote accessible graph view to %s\n", outPath)
		return 0
	}
	if args[0] == "domain-map" {
		parsedDir, graphPath, outPath, labels, history, model, ok := parseCitationsDomainMap(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge citations domain-map --parsed-dir <dir> --out <domain-map.json> [--graph <graph.json> --label topic=label --history action:topic1,topic2:result:reviewer:reason --model <name>]")
		}
		docs, err := readParsedDocuments(parsedDir)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "domain_map_parsed_read_failed", err.Error())
		}
		var graphData []byte
		if graphPath != "" {
			graphData, err = os.ReadFile(graphPath)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "domain_map_graph_read_failed", err.Error())
			}
		}
		artifact, err := citations.BuildDomainMapArtifact(docs, graphData, citations.DomainMapOptions{ReviewerLabels: labels, MergeSplitHistory: history, ModelSettings: citations.DomainMapModelSettings{Model: model, EmbeddingProvider: "deterministic-keyword", MinTopicSize: 1}})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "domain_map_failed", err.Error())
		}
		if err := writeJSONFile(outPath, artifact); err != nil {
			return writeError(stdout, stderr, opts, 1, "domain_map_write_failed", err.Error())
		}
		if opts.Project != "" {
			now := time.Now().UTC()
			_ = provenance.Append(opts.Project, provenance.Event{SchemaVersion: "1", ID: "evt_" + now.Format("20060102T150405Z") + "_domain_map", Timestamp: now.Format(time.RFC3339), Actor: "rforge", Action: "citations.domain_map.created", Target: outPath, Inputs: map[string]any{"parsedDir": parsedDir, "graph": graphPath}, Outputs: map[string]any{"topics": len(artifact.Topics), "path": outPath}})
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"domainMap": artifact, "path": outPath})
		}
		fmt.Fprintf(stdout, "wrote domain map to %s\n", outPath)
		return 0
	}
	if args[0] == "import-bibliography" {
		parsedPath, parsedDir, outPath, reportPath, evidencePath, ok := parseCitationsImportBibliography(args[1:], opts.Project)
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge citations import-bibliography (--parsed <parsed.json>|--parsed-dir <dir>) --out <graph.json> --report <report.json> [--evidence <evidence.json>]")
		}
		docs := []parsing.ParsedDocument{}
		if parsedDir != "" {
			var err error
			docs, err = readParsedDocuments(parsedDir)
			if err != nil {
				return writeError(stdout, stderr, opts, 1, "citation_bibliography_read_failed", err.Error())
			}
		} else {
			var doc parsing.ParsedDocument
			if err := readJSONFile(parsedPath, &doc); err != nil {
				return writeError(stdout, stderr, opts, 1, "citation_bibliography_read_failed", err.Error())
			}
			docs = append(docs, doc)
		}
		var items []evidence.EvidenceItem
		if evidencePath != "" {
			_ = readJSONFile(evidencePath, &items)
		}
		report := citations.ImportParsedBibliographies(docs, items)
		graphData, err := report.Graph.ExportJSON()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_bibliography_export_failed", err.Error())
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_bibliography_write_failed", err.Error())
		}
		if err := os.WriteFile(outPath, graphData, 0o644); err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_bibliography_write_failed", err.Error())
		}
		if err := writeJSONFile(reportPath, report); err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_bibliography_report_write_failed", err.Error())
		}
		if opts.Project != "" {
			now := time.Now().UTC()
			if err := provenance.Append(opts.Project, provenance.Event{SchemaVersion: "1", ID: "evt_" + now.Format("20060102T150405Z") + "_bibliography_import", Timestamp: now.Format(time.RFC3339), Actor: "rforge", Action: "citations.bibliography.imported", Target: outPath, Inputs: map[string]any{"parsed": parsedPath, "parsedDir": parsedDir, "evidence": evidencePath}, Outputs: map[string]any{"graph": outPath, "report": reportPath, "edges": report.EdgeCount}}); err != nil {
				return writeError(stdout, stderr, opts, 1, "citation_bibliography_provenance_failed", err.Error())
			}
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"bibliographyImport": report, "graphPath": outPath, "reportPath": reportPath})
		}
		fmt.Fprintf(stdout, "imported %d bibliography edges to %s\n", report.EdgeCount, outPath)
		return 0
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
	source, paperID, direction, out, runStatePath, limit, depth, maxRecords, maxAPICalls, retryBudget, resumeCursor, dryRun, importLibrary, ok := parseCitationsExpand(args[1:])
	if !ok || (source != "semantic-scholar" && source != "openalex" && source != "crossref") {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge citations expand --source semantic-scholar|openalex|crossref --paper <id> --direction <references|citations|both> --out <file>")
	}
	budget := sources.NormalizeGraphExpansionBudget(sources.GraphExpansionBudget{MaxDepth: depth, MaxNodes: maxRecords, MaxAPICalls: maxAPICalls, RetryBudget: retryBudget, ResumeCursor: resumeCursor})
	budgetEstimate := sources.EstimateGraphExpansionBudget(source, paperID, sources.SemanticScholarGraphDirection(direction), limit, budget)
	if dryRun {
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"budgetEstimate": budgetEstimate, "dryRun": true})
		}
		fmt.Fprintln(stdout, budgetEstimate.DryRunPlan)
		return 0
	}
	var expansion sources.CitationGraphExpansion
	var err error
	if source == "openalex" {
		baseURL := os.Getenv("RFORGE_OPENALEX_URL")
		if baseURL == "" {
			baseURL = "https://api.openalex.org"
		}
		expansion, err = sources.NewOpenAlexConnector(defaultSourceHTTPClient(baseURL)).ExpandCitationGraph(context.Background(), sources.OpenAlexGraphQuery{WorkID: paperID, Direction: sources.SemanticScholarGraphDirection(direction), Limit: limit})
	} else if source == "crossref" {
		expansion, err = expandCrossrefReferences(context.Background(), paperID)
	} else {
		connector := sources.NewSemanticScholarConnector(defaultSemanticScholarHTTPClient())
		expansion, err = expandSemanticScholarRecursive(context.Background(), connector, paperID, sources.SemanticScholarGraphDirection(direction), limit, budget.MaxDepth, budget.MaxNodes, maxAPICalls)
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
	if runStatePath != "" && source == "semantic-scholar" {
		run := sources.NewSemanticScholarGraphRun(sources.SemanticScholarGraphRunOptions{SeedID: paperID, Direction: sources.SemanticScholarGraphDirection(direction), Limit: limit, Depth: budget.MaxDepth, MaxRecords: budget.MaxNodes, RequestedFields: []string{"paperId", "title", "abstract", "year", "venue", "externalIds"}, Budget: budget})
		run = run.RecordExpansion(expansion, nil, 0)
		if err := writeJSONFile(runStatePath, run); err != nil {
			return writeError(stdout, stderr, opts, 1, "citation_run_state_write_failed", err.Error())
		}
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
				"source":       source,
				"paper":        paperID,
				"direction":    direction,
				"limit":        limit,
				"depth":        depth,
				"maxRecords":   budget.MaxNodes,
				"maxApiCalls":  budget.MaxAPICalls,
				"retryBudget":  budget.RetryBudget,
				"resumeCursor": budget.ResumeCursor,
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
		return writeJSON(stdout, 0, map[string]any{"path": out, "runState": runStatePath, "edges": len(expansion.Edges), "rawRef": expansion.RawRef, "imported": imported, "depth": budget.MaxDepth, "maxRecords": budget.MaxNodes, "budgetEstimate": budgetEstimate})
	}
	fmt.Fprintf(stdout, "wrote citation graph with %d edges to %s\n", len(expansion.Edges), out)
	return 0
}

func expandCrossrefReferences(ctx context.Context, doi string) (sources.CitationGraphExpansion, error) {
	baseURL := os.Getenv("RFORGE_CROSSREF_URL")
	if baseURL == "" {
		baseURL = "https://api.crossref.org"
	}
	response, err := sources.NewCrossrefConnector(defaultSourceHTTPClient(baseURL)).References(ctx, doi)
	if err != nil {
		return sources.CitationGraphExpansion{}, err
	}
	seed := strings.ToLower(strings.TrimSpace(doi))
	expansion := sources.CitationGraphExpansion{SeedID: seed, Records: map[string]sources.SourceRecord{}, RawRef: response.RawRef}
	for _, record := range response.Records {
		refID := strings.TrimSpace(record.Identifiers.DOI)
		if refID == "" {
			refID = strings.TrimSpace(record.SourceID)
		}
		if refID == "" {
			continue
		}
		expansion.Edges = append(expansion.Edges, sources.CitationEdge{SourceID: seed, TargetID: refID})
		expansion.Records[refID] = record
	}
	return expansion, nil
}

func expandSemanticScholarRecursive(ctx context.Context, connector sources.SemanticScholarConnector, seedID string, direction sources.SemanticScholarGraphDirection, limit, depth, maxRecords, maxAPICalls int) (sources.CitationGraphExpansion, error) {
	if depth <= 0 {
		depth = 1
	}
	aggregate := sources.CitationGraphExpansion{SeedID: seedID, Records: map[string]sources.SourceRecord{}, RawRef: fmt.Sprintf("semantic-scholar:/recursive?seed=%s&direction=%s&limit=%d&depth=%d&max_records=%d", seedID, direction, limit, depth, maxRecords)}
	visited := map[string]bool{}
	seenEdges := map[string]bool{}
	frontier := []string{seedID}
	apiCalls := 0
	for level := 0; level < depth && len(frontier) > 0; level++ {
		nextSet := map[string]bool{}
		for _, paperID := range frontier {
			if visited[paperID] {
				continue
			}
			visited[paperID] = true
			if maxAPICalls > 0 && apiCalls >= maxAPICalls {
				aggregate.RawRef += "&budget_stopped=max_api_calls"
				return aggregate, nil
			}
			apiCalls++
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
	if len(args) == 1 && args[0] == "acquisition-queue" {
		if opts.Project == "" {
			return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for oa acquisition-queue")
		}
		store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_open_failed", err.Error())
		}
		records, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", err.Error())
		}
		queue := documents.BuildLegalAcquisitionQueue(opts.Project, sources.CompareOpenAccessCandidates(records))
		path := acquisitionQueuePath(opts.Project)
		if err := writeJSONFile(path, queue); err != nil {
			return writeError(stdout, stderr, opts, 1, "acquisition_queue_write_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"queue": queue, "path": path})
		}
		fmt.Fprintf(stdout, "wrote %d legal acquisition queue items to %s\n", len(queue.Items), path)
		return 0
	}
	if len(args) > 0 && args[0] == "acquisition-approve" {
		return executeAcquisitionApprove(args[1:], stdout, stderr, opts)
	}
	if len(args) > 0 && args[0] == "privacy-review" {
		return executePrivacyReview(args[1:], stdout, stderr, opts)
	}
	if len(args) > 0 && args[0] == "privacy-approve" {
		return executePrivacyApprove(args[1:], stdout, stderr, opts)
	}
	if len(args) == 1 && args[0] == "candidates" {
		if opts.Project == "" {
			return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for oa candidates")
		}
		store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_open_failed", err.Error())
		}
		records, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", err.Error())
		}
		comparison := sources.CompareOpenAccessCandidates(records)
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"comparison": comparison})
		}
		for _, candidate := range comparison.Candidates {
			fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", candidate.Source, candidate.DOI, candidate.License, candidate.URL)
		}
		return 0
	}
	if len(args) != 2 || args[0] != "lookup" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge oa lookup <doi>|candidates|acquisition-queue|acquisition-approve <id>|privacy-review|privacy-approve")
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

func executePrivacyReview(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for oa privacy-review")
	}
	values, err := parseKeyValueFlags(args, map[string]bool{"--report": true})
	if err != nil {
		return writeError(stdout, stderr, opts, 2, "usage", err.Error())
	}
	store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_open_failed", err.Error())
	}
	records, err := store.List()
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_list_failed", err.Error())
	}
	reportText := ""
	if reportPath := strings.TrimSpace(values["--report"]); reportPath != "" {
		data, err := os.ReadFile(reportPath)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "privacy_report_read_failed", err.Error())
		}
		reportText = string(data)
	}
	review := documents.ReviewPrivacyLicensing(documents.PrivacyLicensingReviewInput{Records: records, ShareableReport: reportText})
	path := privacyReviewPath(opts.Project)
	if err := writeJSONFile(path, review); err != nil {
		return writeError(stdout, stderr, opts, 1, "privacy_review_write_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"review": review, "path": path})
	}
	fmt.Fprintf(stdout, "privacy/licensing review found %d issue(s)\n", len(review.Issues))
	return 0
}

func executePrivacyApprove(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for oa privacy-approve")
	}
	values, err := parseKeyValueFlags(args, map[string]bool{"--reviewer": true, "--reason": true})
	if err != nil {
		return writeError(stdout, stderr, opts, 2, "usage", err.Error())
	}
	path := privacyReviewPath(opts.Project)
	data, err := os.ReadFile(path)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "privacy_review_read_failed", err.Error())
	}
	var review documents.PrivacyLicensingReview
	if err := json.Unmarshal(data, &review); err != nil {
		return writeError(stdout, stderr, opts, 1, "privacy_review_decode_failed", err.Error())
	}
	review = documents.ApprovePrivacyLicensing(review, values["--reviewer"], values["--reason"])
	if err := writeJSONFile(path, review); err != nil {
		return writeError(stdout, stderr, opts, 1, "privacy_review_write_failed", err.Error())
	}
	now := time.Now().UTC()
	_ = provenance.Append(opts.Project, provenance.Event{SchemaVersion: "1", ID: "evt_" + now.Format("20060102T150405Z") + "_privacy_review", Timestamp: now.Format(time.RFC3339), Actor: "rforge", Action: "privacy.licensing.approved", Target: path, Inputs: map[string]any{"reviewer": values["--reviewer"], "reason": values["--reason"]}, Outputs: map[string]any{"issues": len(review.Issues)}, Warnings: []string{}})
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"review": review, "path": path})
	}
	fmt.Fprintln(stdout, "approved privacy/licensing review")
	return 0
}

func privacyReviewPath(project string) string {
	return filepath.Join(project, "data", "privacy-licensing-review.json")
}

func executeAcquisitionApprove(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for oa acquisition-approve")
	}
	if len(args) < 1 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> oa acquisition-approve <id> --reviewer <name> --reason <text>")
	}
	values, err := parseKeyValueFlags(args[1:], map[string]bool{"--reviewer": true, "--reason": true})
	if err != nil {
		return writeError(stdout, stderr, opts, 2, "usage", err.Error())
	}
	path := acquisitionQueuePath(opts.Project)
	var queue documents.LegalAcquisitionQueue
	data, err := os.ReadFile(path)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "acquisition_queue_read_failed", err.Error())
	}
	if err := json.Unmarshal(data, &queue); err != nil {
		return writeError(stdout, stderr, opts, 1, "acquisition_queue_decode_failed", err.Error())
	}
	approved := false
	for i := range queue.Items {
		if queue.Items[i].ID == args[0] {
			queue.Items[i] = documents.ApproveAcquisition(queue.Items[i], values["--reviewer"], values["--reason"])
			approved = true
		}
	}
	if !approved {
		return writeError(stdout, stderr, opts, 2, "acquisition_queue_item_not_found", "acquisition queue item not found")
	}
	if err := writeJSONFile(path, queue); err != nil {
		return writeError(stdout, stderr, opts, 1, "acquisition_queue_write_failed", err.Error())
	}
	now := time.Now().UTC()
	_ = provenance.Append(opts.Project, provenance.Event{SchemaVersion: "1", ID: "evt_" + now.Format("20060102T150405Z") + "_document_acquisition", Timestamp: now.Format(time.RFC3339), Actor: "rforge", Action: "document.acquisition.approved", Target: args[0], Inputs: map[string]any{"reviewer": values["--reviewer"], "reason": values["--reason"]}, Outputs: map[string]any{"queue": path}, Warnings: []string{}})
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"queue": queue, "path": path})
	}
	fmt.Fprintf(stdout, "approved acquisition queue item %s\n", args[0])
	return 0
}

func acquisitionQueuePath(project string) string {
	return filepath.Join(project, "data", "legal-acquisition-queue.json")
}

func executeSearch(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) > 0 && args[0] == "import" {
		return executeSearchImport(args[1:], stdout, stderr, opts)
	}
	if len(args) > 0 && args[0] == "batch" {
		return executeSearchBatch(args[1:], stdout, stderr, opts)
	}
	if len(args) > 0 && args[0] == "related" {
		return executeSearchRelated(args[1:], stdout, stderr, opts)
	}
	if len(args) > 0 && args[0] == "stats" {
		return executeSearchStats(args[1:], stdout, stderr, opts)
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

type searchBatchOptions struct {
	Queries         []string
	QueriesFile     string
	Sources         []string
	Limit           int
	OutDir          string
	ContinueOnError bool
	WriteStats      bool
}

type searchBatchFailure struct {
	Source string `json:"source"`
	Query  string `json:"query"`
	Error  string `json:"error"`
}

type searchBatchManifest struct {
	SchemaVersion string   `json:"schemaVersion"`
	CreatedAt     string   `json:"createdAt"`
	Queries       []string `json:"queries"`
	Sources       []string `json:"sources"`
	Limit         int      `json:"limit"`
	Results       int      `json:"results"`
	Deduped       int      `json:"deduped"`
	Failures      int      `json:"failures"`
}

func executeSearchBatch(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	batch, ok := parseSearchBatch(args)
	if !ok {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge search batch --queries <file> --sources openalex,crossref --out <dir> [--query <query>] [--limit N] [--continue-on-error] [--stats]")
	}
	queries := append([]string{}, batch.Queries...)
	if batch.QueriesFile != "" {
		fileQueries, err := readSearchBatchQueries(batch.QueriesFile)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "search_batch_queries_failed", err.Error())
		}
		queries = append(queries, fileQueries...)
	}
	queries = uniqueNonEmptyStrings(queries)
	if len(queries) == 0 || len(batch.Sources) == 0 || strings.TrimSpace(batch.OutDir) == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge search batch --queries <file> --sources openalex,crossref --out <dir> [--query <query>] [--limit N] [--continue-on-error] [--stats]")
	}
	if err := os.MkdirAll(filepath.Join(batch.OutDir, "raw"), 0o755); err != nil {
		return writeError(stdout, stderr, opts, 1, "search_batch_out_failed", err.Error())
	}
	resultsPath := filepath.Join(batch.OutDir, "results.jsonl")
	dedupedPath := filepath.Join(batch.OutDir, "results-deduped.jsonl")
	failuresPath := filepath.Join(batch.OutDir, "failures.jsonl")
	resultsFile, err := os.Create(resultsPath)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "search_batch_write_failed", err.Error())
	}
	defer resultsFile.Close()
	failuresFile, err := os.Create(failuresPath)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "search_batch_write_failed", err.Error())
	}
	defer failuresFile.Close()

	allPapers := []library.PaperRecord{}
	failures := []searchBatchFailure{}
	for qi, query := range queries {
		for _, source := range batch.Sources {
			connector, ok := searchConnector(source)
			if !ok {
				failure := searchBatchFailure{Source: source, Query: query, Error: "unknown source"}
				failures = append(failures, failure)
				_ = writeJSONLine(failuresFile, failure)
				if !batch.ContinueOnError {
					return writeError(stdout, stderr, opts, 2, "unknown_source", fmt.Sprintf("unknown source %q", source))
				}
				continue
			}
			response, err := connector.Search(context.Background(), sources.SourceQuery{Terms: query, Limit: batch.Limit, Filters: map[string]string{}})
			if err != nil {
				failure := searchBatchFailure{Source: source, Query: query, Error: err.Error()}
				failures = append(failures, failure)
				_ = writeJSONLine(failuresFile, failure)
				if !batch.ContinueOnError {
					return writeError(stdout, stderr, opts, 1, "search_batch_failed", fmt.Sprintf("%s %q: %v", source, query, err))
				}
				continue
			}
			papers, err := sources.PaperRecords(response)
			if err != nil {
				failure := searchBatchFailure{Source: source, Query: query, Error: err.Error()}
				failures = append(failures, failure)
				_ = writeJSONLine(failuresFile, failure)
				if !batch.ContinueOnError {
					return writeError(stdout, stderr, opts, 1, "search_batch_normalize_failed", fmt.Sprintf("%s %q: %v", source, query, err))
				}
				continue
			}
			rawName := fmt.Sprintf("search-%s-%03d-%s.txt", source, qi+1, slugifySearchBatch(query))
			if err := writeSearchBatchRaw(filepath.Join(batch.OutDir, "raw", rawName), papers); err != nil {
				return writeError(stdout, stderr, opts, 1, "search_batch_write_failed", err.Error())
			}
			for _, paper := range papers {
				allPapers = append(allPapers, paper)
				if err := writeJSONLine(resultsFile, paper); err != nil {
					return writeError(stdout, stderr, opts, 1, "search_batch_write_failed", err.Error())
				}
			}
		}
	}
	deduped := dedupeSearchBatchPapers(allPapers)
	if err := writeSearchBatchJSONL(dedupedPath, deduped); err != nil {
		return writeError(stdout, stderr, opts, 1, "search_batch_write_failed", err.Error())
	}
	if err := writeSearchBatchMarkdown(filepath.Join(batch.OutDir, "results.md"), deduped, failures); err != nil {
		return writeError(stdout, stderr, opts, 1, "search_batch_write_failed", err.Error())
	}
	manifest := searchBatchManifest{SchemaVersion: "1", CreatedAt: time.Now().UTC().Format(time.RFC3339), Queries: queries, Sources: batch.Sources, Limit: batch.Limit, Results: len(allPapers), Deduped: len(deduped), Failures: len(failures)}
	if err := writeJSONFile(filepath.Join(batch.OutDir, "manifest.json"), manifest); err != nil {
		return writeError(stdout, stderr, opts, 1, "search_batch_write_failed", err.Error())
	}
	if batch.WriteStats {
		if err := writeSearchBatchStats(filepath.Join(batch.OutDir, "search-stats.txt"), batch.Sources, len(queries), len(allPapers), len(deduped), failures); err != nil {
			return writeError(stdout, stderr, opts, 1, "search_batch_write_failed", err.Error())
		}
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"out": batch.OutDir, "results": len(allPapers), "deduped": len(deduped), "failures": len(failures), "manifest": filepath.Join(batch.OutDir, "manifest.json")})
	}
	fmt.Fprintf(stdout, "searched %d querie(s) across %d source(s): %d records, %d deduped, %d failure(s)\n", len(queries), len(batch.Sources), len(allPapers), len(deduped), len(failures))
	fmt.Fprintf(stdout, "wrote %s\n", batch.OutDir)
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
	case "concepts":
		entities, rawRef, err = connector.SearchConcepts(context.Background(), sources.SourceQuery{Terms: query, Limit: limit})
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge search --source openalex --query <query> --entity authors|institutions|concepts")
	}
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "search_failed", fmt.Sprintf("search: %v", err))
	}
	disambiguation := sources.BuildOpenAlexDisambiguationQueue(query, entity, entities, rawRef)
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"entities": entities, "disambiguationQueue": disambiguation, "source": "openalex", "entity": entity, "rawRef": rawRef})
	}
	for _, entity := range entities {
		fmt.Fprintf(stdout, "%s\t%s\t%d\n", entity.SourceID, entity.DisplayName, entity.WorksCount)
	}
	return 0
}

func executeSearchStats(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	dir := ""
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--dir" {
			dir = args[i+1]
		}
	}
	if dir == "" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge search stats --dir <dir>")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "stats_read_failed", fmt.Sprintf("read dir: %v", err))
	}
	sourceCounts := map[string]int{}
	sourceFiles := map[string]int{}
	uniqueDOIs := map[string]struct{}{}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "search-") || !strings.HasSuffix(name, ".txt") {
			continue
		}
		source := searchFileSource(name)
		data, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil {
			continue
		}
		sourceFiles[source]++
		count := 0
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 2)
			doi := strings.TrimSpace(parts[0])
			if doi != "" {
				count++
				uniqueDOIs[doi] = struct{}{}
			}
		}
		sourceCounts[source] += count
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{
			"sources":         sourceCounts,
			"sourceFiles":     sourceFiles,
			"totalUniqueDOIs": len(uniqueDOIs),
		})
	}
	// collect and sort source names for deterministic output
	names := make([]string, 0, len(sourceCounts))
	for src := range sourceCounts {
		names = append(names, src)
	}
	sort.Strings(names)
	fmt.Fprintf(stdout, "Source coverage for %s\n", dir)
	for _, src := range names {
		files := sourceFiles[src]
		fmt.Fprintf(stdout, "  %-24s %d records (%d files)\n", src, sourceCounts[src], files)
	}
	fmt.Fprintf(stdout, "\nTotal unique DOIs: %d\n", len(uniqueDOIs))
	return 0
}

func searchFileSource(filename string) string {
	// search-openalex-some-query.txt → openalex
	// search-semantic-scholar-some-query.txt → semantic-scholar
	name := strings.TrimPrefix(filename, "search-")
	name = strings.TrimSuffix(name, ".txt")
	knownSources := []string{"semantic-scholar", "inspire-hep", "openalex", "crossref", "clinicaltrials", "arxiv", "pubmed", "europepmc", "biorxiv", "core", "doaj", "nasa-ads", "zenodo", "dblp", "osf", "opencitations", "base", "zbmath", "figshare", "datacite", "lens", "eric", "hal", "dimensions", "pubchem"}
	for _, src := range knownSources {
		if strings.HasPrefix(name, src) {
			return src
		}
	}
	// fallback: first hyphen-separated segment
	if idx := strings.Index(name, "-"); idx > 0 {
		return name[:idx]
	}
	return name
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
	source, query, pages, limit, filters, resumeStatePath, ok := parseSearchImport(args)
	if !ok || source != "openalex" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> search import --source openalex --query <query> --pages N [--limit N] [--filter source-filter] [--resume-state state.json]")
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
	state, err := loadOpenAlexImportState(resumeStatePath)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "search_import_resume_state_failed", err.Error())
	}
	if err := validateOpenAlexImportState(state, source, query, limit, filters); err != nil {
		return writeError(stdout, stderr, opts, 1, "search_import_resume_state_failed", err.Error())
	}
	if strings.TrimSpace(state.NextCursor) != "" {
		cursor = strings.TrimSpace(state.NextCursor)
	}
	savedNextCursor := strings.TrimSpace(state.NextCursor)
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
		savedNextCursor = strings.TrimSpace(response.NextPageCursor)
		if err := saveOpenAlexImportState(resumeStatePath, openAlexImportState{Source: source, Query: query, Filters: filters, Limit: limit, NextCursor: savedNextCursor, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}); err != nil {
			return writeError(stdout, stderr, opts, 1, "search_import_resume_state_failed", err.Error())
		}
		if savedNextCursor == "" {
			break
		}
		cursor = savedNextCursor
	}
	if err := recordDuplicateEvent(opts.Project, "search.import", map[string]any{"source": source, "query": query, "pages": pages, "limit": limit, "filters": filters, "resumeState": resumeStatePath}, map[string]any{"imported": imported, "skippedDuplicate": skippedDuplicate, "skippedNoIdentifier": skippedNoIdentifier, "rawRefs": rawRefs, "nextCursor": savedNextCursor}); err != nil {
		return writeError(stdout, stderr, opts, 1, "search_import_provenance_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"source": source, "imported": imported, "skippedDuplicate": skippedDuplicate, "skippedNoIdentifier": skippedNoIdentifier, "rawRefs": rawRefs, "resumeState": resumeStatePath, "nextCursor": savedNextCursor})
	}
	fmt.Fprintf(stdout, "imported %d records from %s\n", imported, source)
	return 0
}

type openAlexImportState struct {
	Source     string            `json:"source"`
	Query      string            `json:"query"`
	Filters    map[string]string `json:"filters,omitempty"`
	Limit      int               `json:"limit"`
	NextCursor string            `json:"nextCursor,omitempty"`
	UpdatedAt  string            `json:"updatedAt"`
}

func validateOpenAlexImportState(state openAlexImportState, source, query string, limit int, filters map[string]string) error {
	if strings.TrimSpace(state.Source) == "" {
		return nil
	}
	if state.Source != source || state.Query != query || state.Limit != limit || !sameStringMap(state.Filters, filters) {
		return fmt.Errorf("resume state does not match requested source/query/limit/filters")
	}
	return nil
}

func sameStringMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, av := range a {
		if b[key] != av {
			return false
		}
	}
	return true
}

func loadOpenAlexImportState(path string) (openAlexImportState, error) {
	if strings.TrimSpace(path) == "" {
		return openAlexImportState{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return openAlexImportState{}, nil
		}
		return openAlexImportState{}, err
	}
	var state openAlexImportState
	if err := json.Unmarshal(data, &state); err != nil {
		return openAlexImportState{}, err
	}
	return state, nil
}

func saveOpenAlexImportState(path string, state openAlexImportState) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
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
		// arXiv holds connections ~10s before returning 429; use 30s timeout so the
		// exponential backoff can handle rate limiting properly.
		return sources.NewArXivConnector(defaultArXivHTTPClient(baseURL)), true
	case "crossref":
		baseURL := os.Getenv("RFORGE_CROSSREF_URL")
		if baseURL == "" {
			baseURL = "https://api.crossref.org"
		}
		return sources.NewCrossrefConnector(defaultSourceHTTPClient(baseURL)), true
	case "semantic-scholar":
		return sources.NewSemanticScholarConnector(defaultSemanticScholarHTTPClient()), true
	case "ads":
		baseURL := os.Getenv("RFORGE_ADS_URL")
		if baseURL == "" {
			baseURL = "https://api.adsabs.harvard.edu"
		}
		return sources.NewNASAADSConnector(sources.NewNASAADSHTTPClient(baseURL, os.Getenv("RFORGE_ADS_TOKEN"))), true
	case "europepmc":
		baseURL := os.Getenv("RFORGE_EUROPEPMC_URL")
		if baseURL == "" {
			baseURL = "https://www.ebi.ac.uk/europepmc"
		}
		return sources.NewEuropePMCConnector(defaultSourceHTTPClient(baseURL)), true
	case "doaj":
		baseURL := os.Getenv("RFORGE_DOAJ_URL")
		if baseURL == "" {
			baseURL = "https://doaj.org"
		}
		return sources.NewDOAJConnector(defaultSourceHTTPClient(baseURL)), true
	case "core":
		baseURL := os.Getenv("RFORGE_CORE_URL")
		if baseURL == "" {
			baseURL = "https://api.core.ac.uk"
		}
		return sources.NewCOREConnector(defaultSourceHTTPClient(baseURL)), true
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
	case "zenodo":
		baseURL := os.Getenv("RFORGE_ZENODO_URL")
		if baseURL == "" {
			baseURL = "https://zenodo.org"
		}
		return sources.NewZenodoConnector(defaultSourceHTTPClient(baseURL)), true
	case "inspire-hep":
		baseURL := os.Getenv("RFORGE_INSPIRE_HEP_URL")
		if baseURL == "" {
			baseURL = "https://inspirehep.net"
		}
		return sources.NewInspireHEPConnector(defaultSourceHTTPClient(baseURL)), true
	case "dblp":
		baseURL := os.Getenv("RFORGE_DBLP_URL")
		if baseURL == "" {
			baseURL = "https://dblp.org"
		}
		return sources.NewDBLPConnector(defaultSourceHTTPClient(baseURL)), true
	case "clinicaltrials":
		baseURL := os.Getenv("RFORGE_CLINICALTRIALS_URL")
		if baseURL == "" {
			baseURL = "https://clinicaltrials.gov"
		}
		return sources.NewClinicalTrialsConnector(defaultSourceHTTPClient(baseURL)), true
	case "osf":
		baseURL := os.Getenv("RFORGE_OSF_URL")
		if baseURL == "" {
			baseURL = "https://api.osf.io"
		}
		return sources.NewOSFConnector(defaultSourceHTTPClient(baseURL)), true
	case "biorxiv":
		baseURL := os.Getenv("RFORGE_BIORXIV_URL")
		if baseURL == "" {
			baseURL = "https://api.biorxiv.org"
		}
		return sources.NewBioRxivConnector(defaultSourceHTTPClient(baseURL)), true
	case "opencitations":
		baseURL := os.Getenv("RFORGE_OPENCITATIONS_URL")
		if baseURL == "" {
			baseURL = "https://api.opencitations.net"
		}
		return sources.NewOpenCitationsConnector(defaultSourceHTTPClient(baseURL)), true
	case "base":
		baseURL := os.Getenv("RFORGE_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.base-search.net"
		}
		return sources.NewBASEConnector(defaultSourceHTTPClient(baseURL)), true
	case "zbmath":
		baseURL := os.Getenv("RFORGE_ZBMATH_URL")
		if baseURL == "" {
			baseURL = "https://api.zbmath.org"
		}
		return sources.NewZbMATHConnector(defaultSourceHTTPClient(baseURL)), true
	case "figshare":
		baseURL := os.Getenv("RFORGE_FIGSHARE_URL")
		if baseURL == "" {
			baseURL = "https://api.figshare.com"
		}
		return sources.NewFigshareConnector(defaultSourceHTTPClient(baseURL)), true
	case "datacite":
		baseURL := os.Getenv("RFORGE_DATACITE_URL")
		if baseURL == "" {
			baseURL = "https://api.datacite.org"
		}
		return sources.NewDataCiteConnector(defaultSourceHTTPClient(baseURL)), true
	case "lens":
		baseURL := os.Getenv("RFORGE_LENS_URL")
		if baseURL == "" {
			baseURL = "https://api.lens.org"
		}
		token := os.Getenv("RFORGE_LENS_TOKEN")
		headers := map[string]string{}
		if token != "" {
			headers["Authorization"] = "Bearer " + token
		}
		return sources.NewLensConnector(sources.NewHTTPClient(sources.HTTPClientOptions{
			BaseURL:    baseURL,
			UserAgent:  "ResearchForge/dev",
			Timeout:    15 * time.Second,
			MaxRetries: 2,
			Headers:    headers,
		})), true
	case "eric":
		baseURL := os.Getenv("RFORGE_ERIC_URL")
		if baseURL == "" {
			baseURL = "https://api.ies.ed.gov/eric"
		}
		return sources.NewERICConnector(defaultSourceHTTPClient(baseURL)), true
	case "hal":
		baseURL := os.Getenv("RFORGE_HAL_URL")
		if baseURL == "" {
			baseURL = "https://api.archives-ouvertes.fr"
		}
		return sources.NewHALConnector(defaultSourceHTTPClient(baseURL)), true
	case "dimensions":
		baseURL := os.Getenv("RFORGE_DIMENSIONS_URL")
		if baseURL == "" {
			baseURL = "https://app.dimensions.ai"
		}
		token := os.Getenv("RFORGE_DIMENSIONS_TOKEN")
		headers := map[string]string{}
		if token != "" {
			headers["Authorization"] = "JWT " + token
		}
		return sources.NewDimensionsConnector(sources.NewHTTPClient(sources.HTTPClientOptions{
			BaseURL:    baseURL,
			UserAgent:  "ResearchForge/dev",
			Timeout:    15 * time.Second,
			MaxRetries: 2,
			Headers:    headers,
		})), true
	case "pubchem":
		baseURL := os.Getenv("RFORGE_PUBCHEM_URL")
		if baseURL == "" {
			baseURL = "https://pubchem.ncbi.nlm.nih.gov"
		}
		return sources.NewPubChemConnector(defaultSourceHTTPClient(baseURL)), true
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

func defaultArXivHTTPClient(baseURL string) sources.HTTPClient {
	return sources.NewHTTPClient(sources.HTTPClientOptions{
		BaseURL:    baseURL,
		UserAgent:  "ResearchForge/dev",
		Timeout:    30 * time.Second,
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
		MaxRetries: envInt("RFORGE_SEMANTIC_SCHOLAR_MAX_RETRIES", 3),
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

func parseCitationsAccessibleView(args []string) (string, string, string, string, string, bool) {
	values := map[string]string{"--format": "markdown"}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--graph", "--domain-map", "--out", "--filter", "--format":
			if i+1 >= len(args) {
				return "", "", "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", "", "", false
		}
	}
	format := values["--format"]
	okFormat := format == "markdown" || format == "json"
	return values["--graph"], values["--domain-map"], values["--out"], values["--filter"], format, okFormat && values["--graph"] != "" && values["--out"] != ""
}

func parseCitationsDomainMap(args []string) (string, string, string, map[string]string, []citations.TopicHistoryEvent, string, bool) {
	values := map[string]string{}
	labels := map[string]string{}
	history := []citations.TopicHistoryEvent{}
	model := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--parsed-dir", "--graph", "--out", "--model":
			if i+1 >= len(args) {
				return "", "", "", nil, nil, "", false
			}
			if args[i] == "--model" {
				model = args[i+1]
			} else {
				values[args[i]] = args[i+1]
			}
			i++
		case "--label":
			if i+1 >= len(args) {
				return "", "", "", nil, nil, "", false
			}
			parts := strings.SplitN(args[i+1], "=", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
				return "", "", "", nil, nil, "", false
			}
			labels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			i++
		case "--history":
			if i+1 >= len(args) {
				return "", "", "", nil, nil, "", false
			}
			event, ok := parseTopicHistoryEvent(args[i+1])
			if !ok {
				return "", "", "", nil, nil, "", false
			}
			history = append(history, event)
			i++
		default:
			return "", "", "", nil, nil, "", false
		}
	}
	return values["--parsed-dir"], values["--graph"], values["--out"], labels, history, model, values["--parsed-dir"] != "" && values["--out"] != ""
}

func parseTopicHistoryEvent(value string) (citations.TopicHistoryEvent, bool) {
	parts := strings.SplitN(value, ":", 5)
	if len(parts) < 3 {
		return citations.TopicHistoryEvent{}, false
	}
	topicIDs := []string{}
	for _, topicID := range strings.Split(parts[1], ",") {
		if trimmed := strings.TrimSpace(topicID); trimmed != "" {
			topicIDs = append(topicIDs, trimmed)
		}
	}
	if strings.TrimSpace(parts[0]) == "" || len(topicIDs) == 0 || strings.TrimSpace(parts[2]) == "" {
		return citations.TopicHistoryEvent{}, false
	}
	event := citations.TopicHistoryEvent{Action: strings.TrimSpace(parts[0]), TopicIDs: topicIDs, ResultTopicID: strings.TrimSpace(parts[2])}
	if len(parts) > 3 {
		event.Reviewer = strings.TrimSpace(parts[3])
	}
	if len(parts) > 4 {
		event.Reason = strings.TrimSpace(parts[4])
	}
	return event, true
}

func parseCitationsImportBibliography(args []string, project string) (string, string, string, string, string, bool) {
	values := map[string]string{}
	if project != "" {
		values["--evidence"] = evidenceItemsPath(project)
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--parsed", "--parsed-dir", "--out", "--report", "--evidence":
			if i+1 >= len(args) {
				return "", "", "", "", "", false
			}
			values[args[i]] = args[i+1]
			i++
		default:
			return "", "", "", "", "", false
		}
	}
	parsedOK := (values["--parsed"] != "") != (values["--parsed-dir"] != "")
	return values["--parsed"], values["--parsed-dir"], values["--out"], values["--report"], values["--evidence"], parsedOK && values["--out"] != "" && values["--report"] != ""
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

func parseCitationsExpand(args []string) (string, string, string, string, string, int, int, int, int, int, string, bool, bool, bool) {
	values := map[string]string{}
	limit := 25
	depth := 1
	maxRecords := 0
	maxAPICalls := 0
	retryBudget := 0
	importLibrary := false
	dryRun := false
	fail := func() (string, string, string, string, string, int, int, int, int, int, string, bool, bool, bool) {
		return "", "", "", "", "", 0, 0, 0, 0, 0, "", false, false, false
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--import-library":
			importLibrary = true
		case "--dry-run":
			dryRun = true
		case "--source", "--paper", "--direction", "--out", "--run-state", "--resume-cursor":
			if i+1 >= len(args) {
				return fail()
			}
			values[args[i]] = args[i+1]
			i++
		case "--limit", "--depth", "--max-records", "--max-api-calls", "--retry-budget":
			if i+1 >= len(args) {
				return fail()
			}
			parsed, err := strconv.Atoi(args[i+1])
			if err != nil || parsed < 0 || (parsed == 0 && args[i] != "--retry-budget") {
				return fail()
			}
			switch args[i] {
			case "--limit":
				limit = parsed
			case "--depth":
				depth = parsed
			case "--max-records":
				maxRecords = parsed
			case "--max-api-calls":
				maxAPICalls = parsed
			case "--retry-budget":
				retryBudget = parsed
			}
			i++
		default:
			return fail()
		}
	}
	direction := values["--direction"]
	if direction == "" {
		direction = "both"
	}
	validDirection := direction == string(sources.SemanticScholarDirectionReferences) || direction == string(sources.SemanticScholarDirectionCitations) || direction == string(sources.SemanticScholarDirectionBoth)
	return values["--source"], values["--paper"], direction, values["--out"], values["--run-state"], limit, depth, maxRecords, maxAPICalls, retryBudget, values["--resume-cursor"], dryRun, importLibrary, values["--source"] != "" && values["--paper"] != "" && values["--out"] != "" && validDirection
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

func parseSearchImport(args []string) (string, string, int, int, map[string]string, string, bool) {
	limit := 25
	pages := 1
	filters := map[string]string{}
	var source, query, resumeState string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--source":
			if i+1 >= len(args) {
				return "", "", 0, 0, nil, "", false
			}
			source = args[i+1]
			i++
		case "--query":
			if i+1 >= len(args) {
				return "", "", 0, 0, nil, "", false
			}
			query = args[i+1]
			i++
		case "--pages":
			if i+1 >= len(args) {
				return "", "", 0, 0, nil, "", false
			}
			parsed, err := strconv.Atoi(args[i+1])
			if err != nil || parsed <= 0 {
				return "", "", 0, 0, nil, "", false
			}
			pages = parsed
			i++
		case "--limit":
			if i+1 >= len(args) {
				return "", "", 0, 0, nil, "", false
			}
			parsed, err := strconv.Atoi(args[i+1])
			if err != nil || parsed <= 0 {
				return "", "", 0, 0, nil, "", false
			}
			limit = parsed
			i++
		case "--filter":
			if i+1 >= len(args) {
				return "", "", 0, 0, nil, "", false
			}
			filters["filter"] = appendCommaFilter(filters["filter"], args[i+1])
			i++
		case "--preset":
			if i+1 >= len(args) {
				return "", "", 0, 0, nil, "", false
			}
			preset, ok := sources.OpenAlexFilterPreset(args[i+1])
			if !ok {
				return "", "", 0, 0, nil, "", false
			}
			mergeOpenAlexFilters(filters, preset)
			i++
		case "--from-year", "--to-year", "--type", "--open-access", "--concept":
			if i+1 >= len(args) || !appendOpenAlexAdvancedFilter(filters, args[i], args[i+1]) {
				return "", "", 0, 0, nil, "", false
			}
			i++
		case "--resume-state":
			if i+1 >= len(args) {
				return "", "", 0, 0, nil, "", false
			}
			resumeState = args[i+1]
			i++
		default:
			return "", "", 0, 0, nil, "", false
		}
	}
	return source, query, pages, limit, filters, resumeState, source != "" && strings.TrimSpace(query) != ""
}

func parseSearchBatch(args []string) (searchBatchOptions, bool) {
	batch := searchBatchOptions{Limit: 25}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--queries", "--queries-file":
			if i+1 >= len(args) {
				return searchBatchOptions{}, false
			}
			batch.QueriesFile = args[i+1]
			i++
		case "--query":
			if i+1 >= len(args) {
				return searchBatchOptions{}, false
			}
			batch.Queries = append(batch.Queries, args[i+1])
			i++
		case "--sources":
			if i+1 >= len(args) {
				return searchBatchOptions{}, false
			}
			batch.Sources = splitSearchBatchList(args[i+1])
			i++
		case "--limit":
			if i+1 >= len(args) {
				return searchBatchOptions{}, false
			}
			parsed, err := strconv.Atoi(args[i+1])
			if err != nil || parsed <= 0 {
				return searchBatchOptions{}, false
			}
			batch.Limit = parsed
			i++
		case "--out":
			if i+1 >= len(args) {
				return searchBatchOptions{}, false
			}
			batch.OutDir = args[i+1]
			i++
		case "--continue-on-error":
			batch.ContinueOnError = true
		case "--stats":
			batch.WriteStats = true
		case "--dedupe":
			if i+1 >= len(args) {
				return searchBatchOptions{}, false
			}
			// DOI/title dedupe is always enabled for batch output; accept the flag for CLI readability.
			i++
		default:
			return searchBatchOptions{}, false
		}
	}
	return batch, true
}

func readSearchBatchQueries(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	queries := []string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		queries = append(queries, line)
	}
	return queries, nil
}

func splitSearchBatchList(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func slugifySearchBatch(value string) string {
	fields := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(value)), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	if len(fields) == 0 {
		return "query"
	}
	slug := strings.Join(fields, "-")
	if len(slug) > 80 {
		slug = strings.Trim(slug[:80], "-")
	}
	return slug
}

func writeJSONLine(w io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

func writeSearchBatchRaw(path string, papers []library.PaperRecord) error {
	var b strings.Builder
	for _, paper := range papers {
		fmt.Fprintf(&b, "%s\t%s\n", paper.Identifiers.DOI, paper.Title)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeSearchBatchJSONL(path string, papers []library.PaperRecord) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, paper := range papers {
		if err := writeJSONLine(file, paper); err != nil {
			return err
		}
	}
	return nil
}

func dedupeSearchBatchPapers(papers []library.PaperRecord) []library.PaperRecord {
	seen := map[string]struct{}{}
	out := []library.PaperRecord{}
	for _, paper := range papers {
		key := strings.ToLower(strings.TrimSpace(paper.Identifiers.DOI))
		if key == "" {
			key = strings.ToLower(strings.Join(strings.Fields(paper.Title), " "))
		}
		if key == "" {
			key = fmt.Sprintf("record-%d", len(out))
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, paper)
	}
	return out
}

func writeSearchBatchMarkdown(path string, papers []library.PaperRecord, failures []searchBatchFailure) error {
	var b strings.Builder
	b.WriteString("# Search batch results\n\n")
	fmt.Fprintf(&b, "Deduped records: %d\n\n", len(papers))
	for _, paper := range papers {
		fmt.Fprintf(&b, "- %s", paper.Title)
		if doi := strings.TrimSpace(paper.Identifiers.DOI); doi != "" {
			fmt.Fprintf(&b, " — DOI: `%s`", doi)
		}
		b.WriteString("\n")
	}
	if len(failures) > 0 {
		b.WriteString("\n## Failures\n\n")
		for _, failure := range failures {
			fmt.Fprintf(&b, "- %s / %q: %s\n", failure.Source, failure.Query, failure.Error)
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeSearchBatchStats(path string, sources []string, queryCount, total, deduped int, failures []searchBatchFailure) error {
	bySourceFailures := map[string]int{}
	for _, failure := range failures {
		bySourceFailures[failure.Source]++
	}
	var b strings.Builder
	b.WriteString("Search batch stats\n")
	fmt.Fprintf(&b, "Queries: %d\n", queryCount)
	fmt.Fprintf(&b, "Sources: %s\n", strings.Join(sources, ","))
	fmt.Fprintf(&b, "Records: %d\n", total)
	fmt.Fprintf(&b, "Deduped records: %d\n", deduped)
	fmt.Fprintf(&b, "Failures: %d\n", len(failures))
	if len(bySourceFailures) > 0 {
		names := make([]string, 0, len(bySourceFailures))
		for name := range bySourceFailures {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fmt.Fprintf(&b, "  %s failures: %d\n", name, bySourceFailures[name])
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
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
		case "--preset":
			if i+1 >= len(args) {
				return "", "", 0, nil, false
			}
			preset, ok := sources.OpenAlexFilterPreset(args[i+1])
			if !ok {
				return "", "", 0, nil, false
			}
			mergeOpenAlexFilters(filters, preset)
			i++
		case "--from-year", "--to-year", "--type", "--open-access", "--concept":
			if i+1 >= len(args) || !appendOpenAlexAdvancedFilter(filters, args[i], args[i+1]) {
				return "", "", 0, nil, false
			}
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

func mergeOpenAlexFilters(filters, preset map[string]string) {
	for key, value := range preset {
		if key == "filter" {
			filters[key] = appendCommaFilter(filters[key], value)
			continue
		}
		filters[key] = value
	}
}

func appendOpenAlexAdvancedFilter(filters map[string]string, flag, value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	switch flag {
	case "--from-year":
		if _, err := strconv.Atoi(value); err != nil || len(value) != 4 {
			return false
		}
		filters["filter"] = appendCommaFilter(filters["filter"], "from_publication_date:"+value+"-01-01")
	case "--to-year":
		if _, err := strconv.Atoi(value); err != nil || len(value) != 4 {
			return false
		}
		filters["filter"] = appendCommaFilter(filters["filter"], "to_publication_date:"+value+"-12-31")
	case "--type":
		filters["filter"] = appendCommaFilter(filters["filter"], "type:"+value)
	case "--open-access":
		if value != "true" && value != "false" {
			return false
		}
		filters["filter"] = appendCommaFilter(filters["filter"], "is_oa:"+value)
	case "--concept":
		filters["filter"] = appendCommaFilter(filters["filter"], "concepts.id:"+value)
	default:
		return false
	}
	return true
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
