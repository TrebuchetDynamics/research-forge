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

	"github.com/TrebuchetDynamics/research-forge/internal/library"
	"github.com/TrebuchetDynamics/research-forge/internal/provenance"
	"github.com/TrebuchetDynamics/research-forge/internal/sources"
)

func executeImport(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 2 || !supportedImportExportFormat(args[0]) {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> import <json|csv|bibtex|ris|csl-json|zotero-rdf> <file>")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for import commands")
	}
	var records []library.PaperRecord
	var skippedNoIdentifier int
	var err error
	switch args[0] {
	case "csv":
		records, skippedNoIdentifier, err = library.ImportCSV(args[1])
	case "bibtex":
		records, skippedNoIdentifier, err = library.ImportBibTeX(args[1])
	case "ris":
		records, skippedNoIdentifier, err = library.ImportRIS(args[1])
	case "csl-json":
		records, skippedNoIdentifier, err = library.ImportCSLJSON(args[1])
	case "zotero-rdf":
		records, skippedNoIdentifier, err = library.ImportZoteroRDF(args[1])
	default:
		records, skippedNoIdentifier, err = library.ImportJSON(args[1])
	}
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "import_failed", fmt.Sprintf("import %s: %v", args[0], err))
	}
	store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_open_failed", fmt.Sprintf("open library: %v", err))
	}
	summary, err := store.ImportRecords(records)
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "import_store_failed", fmt.Sprintf("store imported records: %v", err))
	}
	skippedNoIdentifier += summary.SkippedNoIdentifier
	if opts.JSON {
		skipped := summary.SkippedDuplicate
		if skipped == nil {
			skipped = []string{}
		}
		return writeJSON(stdout, 0, map[string]any{
			"imported":              summary.Imported,
			"skipped_duplicate":     skipped,
			"skipped_no_identifier": skippedNoIdentifier,
		})
	}
	fmt.Fprintf(stdout, "imported %d records (skipped %d duplicates, %d without identifiers)\n", summary.Imported, len(summary.SkippedDuplicate), skippedNoIdentifier)
	return 0
}

func supportedImportExportFormat(format string) bool {
	return format == "json" || format == "csv" || format == "bibtex" || format == "ris" || format == "csl-json" || format == "zotero-rdf"
}

func executeExport(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) != 2 || !supportedImportExportFormat(args[0]) {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> export <json|csv|bibtex|ris|csl-json|zotero-rdf> <file>")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for export commands")
	}
	store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_open_failed", fmt.Sprintf("open library: %v", err))
	}
	records, err := store.List()
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_list_failed", fmt.Sprintf("list library: %v", err))
	}
	switch args[0] {
	case "csv":
		err = library.ExportCSV(args[1], records)
	case "bibtex":
		err = library.ExportBibTeX(args[1], records)
	case "ris":
		err = library.ExportRIS(args[1], records)
	case "csl-json":
		err = library.ExportCSLJSON(args[1], records)
	case "zotero-rdf":
		err = library.ExportZoteroRDF(args[1], records)
	default:
		err = library.ExportJSON(args[1], records)
	}
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "export_failed", fmt.Sprintf("export %s: %v", args[0], err))
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"exported": len(records), "path": args[1]})
	}
	fmt.Fprintf(stdout, "exported %d records to %s\n", len(records), args[1])
	return 0
}

func executeDuplicate(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> duplicate <report|merge|split>")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for duplicate commands")
	}
	store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_open_failed", fmt.Sprintf("open library: %v", err))
	}
	papers, err := store.List()
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_list_failed", fmt.Sprintf("list library: %v", err))
	}
	switch args[0] {
	case "report":
		sourceFilter, ok := parseDuplicateReportArgs(args[1:])
		if !ok {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> duplicate report [--source <source>]")
		}
		matches := duplicateMatches(papers, sourceFilter)
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"matches": matches})
		}
		for _, match := range matches {
			fmt.Fprintf(stdout, "%s\t%.2f\t%d\t%d\n", match.Reason, match.Score, match.LeftIndex, match.RightIndex)
		}
		return 0
	case "merge":
		if len(args) != 3 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> duplicate merge <left-index> <right-index>")
		}
		leftIndex, rightIndex, ok := parseTwoIndexes(args[1], args[2], len(papers))
		if !ok || leftIndex == rightIndex {
			return writeError(stdout, stderr, opts, 2, "invalid_duplicate_indexes", "duplicate merge indexes must reference two existing records")
		}
		merged := library.MergeDuplicate(papers[leftIndex], papers[rightIndex])
		updated := removePaperIndexes(papers, leftIndex, rightIndex)
		updated = append(updated, merged)
		if err := store.ReplaceAll(updated); err != nil {
			return writeError(stdout, stderr, opts, 1, "duplicate_merge_failed", fmt.Sprintf("merge duplicate: %v", err))
		}
		if err := recordDuplicateEvent(opts.Project, "duplicate.merge", map[string]any{"leftIndex": leftIndex, "rightIndex": rightIndex}, map[string]any{"recordCount": len(updated)}); err != nil {
			return writeError(stdout, stderr, opts, 1, "duplicate_provenance_failed", fmt.Sprintf("record duplicate provenance: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"merged": true, "recordCount": len(updated)})
		}
		fmt.Fprintln(stdout, "merged duplicate records")
		return 0
	case "split":
		if len(args) != 3 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> duplicate split <record-index> <json-file>")
		}
		index, err := strconv.Atoi(args[1])
		if err != nil || index < 0 || index >= len(papers) {
			return writeError(stdout, stderr, opts, 2, "invalid_duplicate_index", "duplicate split index must reference an existing record")
		}
		replacements, _, err := library.ImportJSON(args[2])
		if err != nil || len(replacements) < 2 {
			return writeError(stdout, stderr, opts, 2, "invalid_split_file", "duplicate split requires a JSON file with at least two replacement records")
		}
		updated := append([]library.PaperRecord{}, papers[:index]...)
		updated = append(updated, replacements...)
		updated = append(updated, papers[index+1:]...)
		if err := store.ReplaceAll(updated); err != nil {
			return writeError(stdout, stderr, opts, 1, "duplicate_split_failed", fmt.Sprintf("split duplicate: %v", err))
		}
		if err := recordDuplicateEvent(opts.Project, "duplicate.split", map[string]any{"index": index, "replacementFile": args[2]}, map[string]any{"recordCount": len(updated), "replacementCount": len(replacements)}); err != nil {
			return writeError(stdout, stderr, opts, 1, "duplicate_provenance_failed", fmt.Sprintf("record duplicate provenance: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"split": true, "recordCount": len(updated)})
		}
		fmt.Fprintln(stdout, "split duplicate record")
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> duplicate <report|merge|split>")
	}
}

func parseTwoIndexes(left, right string, max int) (int, int, bool) {
	leftIndex, leftErr := strconv.Atoi(left)
	rightIndex, rightErr := strconv.Atoi(right)
	if leftErr != nil || rightErr != nil || leftIndex < 0 || rightIndex < 0 || leftIndex >= max || rightIndex >= max {
		return 0, 0, false
	}
	return leftIndex, rightIndex, true
}

func removePaperIndexes(papers []library.PaperRecord, leftIndex, rightIndex int) []library.PaperRecord {
	out := make([]library.PaperRecord, 0, len(papers)-2)
	for i, paper := range papers {
		if i != leftIndex && i != rightIndex {
			out = append(out, paper)
		}
	}
	return out
}

func recordDuplicateEvent(projectPath, action string, inputs, outputs map[string]any) error {
	now := time.Now().UTC()
	return provenance.Append(projectPath, provenance.Event{
		SchemaVersion: "1",
		ID:            "evt_" + now.Format("20060102T150405Z") + "_duplicate",
		Timestamp:     now.Format(time.RFC3339),
		Actor:         "rforge",
		Action:        action,
		Target:        projectPath,
		Inputs:        inputs,
		Outputs:       outputs,
		Warnings:      []string{},
	})
}

type duplicateReportMatch struct {
	LeftIndex    int
	RightIndex   int
	Duplicate    bool
	Score        float64
	Reason       string
	LeftSources  []string
	RightSources []string
}

func parseDuplicateReportArgs(args []string) (string, bool) {
	if len(args) == 0 {
		return "", true
	}
	if len(args) == 2 && args[0] == "--source" && strings.TrimSpace(args[1]) != "" {
		return strings.TrimSpace(args[1]), true
	}
	return "", false
}

func duplicateMatches(papers []library.PaperRecord, sourceFilter string) []duplicateReportMatch {
	matches := []duplicateReportMatch{}
	for i := 0; i < len(papers); i++ {
		for j := i + 1; j < len(papers); j++ {
			if sourceFilter != "" && !paperHasSource(papers[i], sourceFilter) && !paperHasSource(papers[j], sourceFilter) {
				continue
			}
			match := library.ScoreDuplicate(papers[i], papers[j])
			if match.Duplicate {
				matches = append(matches, duplicateReportMatch{LeftIndex: i, RightIndex: j, Duplicate: true, Score: match.Score, Reason: match.Reason, LeftSources: paperSources(papers[i]), RightSources: paperSources(papers[j])})
			}
		}
	}
	return matches
}

func paperHasSource(paper library.PaperRecord, source string) bool {
	for _, ref := range paper.SourceRefs {
		if ref.Source == source {
			return true
		}
	}
	return false
}

func paperSources(paper library.PaperRecord) []string {
	sources := []string{}
	seen := map[string]bool{}
	for _, ref := range paper.SourceRefs {
		if ref.Source != "" && !seen[ref.Source] {
			seen[ref.Source] = true
			sources = append(sources, ref.Source)
		}
	}
	return sources
}

func executeLibrary(args []string, stdout, stderr io.Writer, opts globalOptions) int {
	if len(args) == 0 {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> library <list|identity-resolve|reference-manager-matrix|refresh-doi|import-crossref-refs>")
	}
	if opts.Project == "" {
		return writeError(stdout, stderr, opts, 2, "missing_project", "--project is required for library commands")
	}
	store, err := library.OpenStore(filepath.Join(opts.Project, "data", "library.json"))
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_open_failed", fmt.Sprintf("open library: %v", err))
	}
	switch args[0] {
	case "list":
		if len(args) != 1 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> library list")
		}
		papers, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", fmt.Sprintf("list library: %v", err))
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"papers": papers})
		}
		for _, paper := range papers {
			fmt.Fprintf(stdout, "%s\t%s\n", paper.Identifiers.DOI, paper.Title)
		}
		return 0
	case "identity-conflicts":
		if len(args) != 1 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> library identity-conflicts")
		}
		papers, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", fmt.Sprintf("list library: %v", err))
		}
		conflicts := library.DetectIdentityConflicts(library.ResolveIdentityClusters(papers), papers)
		for _, conflict := range conflicts {
			_ = library.AppendIdentityConflict(identityDecisionLogPath(opts.Project), conflict)
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"conflicts": conflicts})
		}
		for _, conflict := range conflicts {
			fmt.Fprintf(stdout, "%s\t%s\t%s\n", conflict.ClusterID, conflict.Severity, conflict.Reason)
		}
		return 0
	case "identity-decision":
		return executeIdentityDecision(args[1:], stdout, stderr, opts, store)
	case "identity-resolve":
		if len(args) != 1 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> library identity-resolve")
		}
		papers, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", fmt.Sprintf("list library: %v", err))
		}
		report := library.ResolveIdentityClusters(papers)
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"report": report})
		}
		fmt.Fprintln(stdout, "Source-fusion identity resolution:")
		for _, cluster := range report.Clusters {
			fmt.Fprintf(stdout, "- %s records=%v identifiers=%v\n", cluster.ID, cluster.RecordIndexes, cluster.Identifiers)
			for _, match := range cluster.Matches {
				fmt.Fprintf(stdout, "  %s: %s=%s (%.2f) %s\n", match.Rule, match.Identifier, match.Value, match.Confidence, match.Explanation)
			}
		}
		return 0
	case "reference-manager-matrix":
		if len(args) != 1 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> library reference-manager-matrix")
		}
		papers, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", fmt.Sprintf("list library: %v", err))
		}
		matrix := library.BuildReferenceManagerInterchangeMatrix(papers)
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"matrix": matrix})
		}
		fmt.Fprintln(stdout, "Reference-manager interchange fidelity matrix:")
		for _, format := range matrix.Formats {
			fmt.Fprintf(stdout, "- %s (%s)\n", format.Label, format.Format)
			for field, fidelity := range format.Fields {
				fmt.Fprintf(stdout, "  %s: %s — %s\n", field, fidelity.Status, fidelity.Note)
			}
		}
		return 0
	case "import-crossref-refs":
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> library import-crossref-refs <doi>")
		}
		baseURL := os.Getenv("RFORGE_CROSSREF_URL")
		if baseURL == "" {
			baseURL = "https://api.crossref.org"
		}
		response, err := sources.NewCrossrefConnector(defaultSourceHTTPClient(baseURL)).References(context.Background(), args[1])
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_crossref_refs_failed", err.Error())
		}
		importable := make([]sources.SourceRecord, 0, len(response.Records))
		for _, record := range response.Records {
			if record.Identifiers.DOI != "" && record.Title != "" {
				importable = append(importable, record)
			}
		}
		papers, err := sources.PaperRecords(sources.SourceResponse{Records: importable, RawRef: response.RawRef})
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_crossref_refs_normalize_failed", err.Error())
		}
		summary, err := store.ImportRecords(papers)
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_crossref_refs_import_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"imported": summary.Imported, "skippedDuplicate": summary.SkippedDuplicate, "skippedNoIdentifier": summary.SkippedNoIdentifier, "extracted": len(response.Records), "importable": len(importable)})
		}
		fmt.Fprintf(stdout, "imported %d Crossref references\n", summary.Imported)
		return 0
	case "refresh-doi":
		if len(args) != 2 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> library refresh-doi <doi>")
		}
		result, err := refreshCrossrefDOIs(store, []string{args[1]})
		if err != nil {
			return writeError(stdout, stderr, opts, result.exitCode, result.code, err.Error())
		}
		if result.Refreshed == 0 {
			return writeError(stdout, stderr, opts, 2, "library_refresh_not_found", "DOI not found in library")
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"refreshed": strings.ToLower(strings.TrimSpace(args[1]))})
		}
		fmt.Fprintf(stdout, "refreshed %s\n", strings.ToLower(strings.TrimSpace(args[1])))
		return 0
	case "refresh-crossref":
		if len(args) != 1 {
			return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> library refresh-crossref")
		}
		papers, err := store.List()
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "library_list_failed", fmt.Sprintf("list library: %v", err))
		}
		dois := []string{}
		skippedNoDOI := 0
		for _, paper := range papers {
			if strings.TrimSpace(paper.Identifiers.DOI) == "" {
				skippedNoDOI++
				continue
			}
			dois = append(dois, paper.Identifiers.DOI)
		}
		result, err := refreshCrossrefDOIs(store, dois)
		if err != nil {
			return writeError(stdout, stderr, opts, result.exitCode, result.code, err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"refreshed": result.Refreshed, "skippedNoDOI": skippedNoDOI})
		}
		fmt.Fprintf(stdout, "refreshed %d Crossref DOI records\n", result.Refreshed)
		return 0
	default:
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> library <list|identity-resolve|reference-manager-matrix|refresh-doi|refresh-crossref|import-crossref-refs>")
	}
}

type crossrefRefreshResult struct {
	Refreshed int
	code      string
	exitCode  int
}

func executeIdentityDecision(args []string, stdout, stderr io.Writer, opts globalOptions, store library.Store) int {
	if len(args) == 1 && args[0] == "log" {
		log, err := library.ReadIdentityDecisionLog(identityDecisionLogPath(opts.Project))
		if err != nil {
			return writeError(stdout, stderr, opts, 1, "identity_decision_log_failed", err.Error())
		}
		if opts.JSON {
			return writeJSON(stdout, 0, map[string]any{"log": log})
		}
		fmt.Fprintf(stdout, "%d identity decisions, %d conflicts\n", len(log.Decisions), len(log.Conflicts))
		return 0
	}
	if len(args) == 0 || args[0] != "record" {
		return writeError(stdout, stderr, opts, 2, "usage", "usage: rforge --project <path> library identity-decision record --action merge|split --cluster <id> --reason <text> --before-indexes <csv> --after-indexes <csv>")
	}
	values, err := parseKeyValueFlags(args[1:], map[string]bool{"--action": true, "--cluster": true, "--reason": true, "--reviewer": true, "--before-indexes": true, "--after-indexes": true})
	if err != nil {
		return writeError(stdout, stderr, opts, 2, "usage", err.Error())
	}
	papers, err := store.List()
	if err != nil {
		return writeError(stdout, stderr, opts, 1, "library_list_failed", fmt.Sprintf("list library: %v", err))
	}
	before, ok := papersByCSVIndexes(papers, values["--before-indexes"])
	if !ok {
		return writeError(stdout, stderr, opts, 2, "invalid_identity_decision_indexes", "before indexes must reference existing records")
	}
	after, ok := papersByCSVIndexes(papers, values["--after-indexes"])
	if !ok {
		return writeError(stdout, stderr, opts, 2, "invalid_identity_decision_indexes", "after indexes must reference existing records")
	}
	decision := library.IdentityDecision{ID: "identity-decision-" + time.Now().UTC().Format("20060102T150405Z"), ClusterID: values["--cluster"], Action: values["--action"], Reviewer: values["--reviewer"], Reason: values["--reason"], Reversible: true, Before: before, After: after}
	if err := library.AppendIdentityDecision(identityDecisionLogPath(opts.Project), decision); err != nil {
		return writeError(stdout, stderr, opts, 1, "identity_decision_record_failed", err.Error())
	}
	if err := recordDuplicateEvent(opts.Project, "identity."+decision.Action+".approved", map[string]any{"clusterId": decision.ClusterID, "reason": decision.Reason}, map[string]any{"reversible": true, "before": len(before), "after": len(after)}); err != nil {
		return writeError(stdout, stderr, opts, 1, "identity_decision_provenance_failed", err.Error())
	}
	if opts.JSON {
		return writeJSON(stdout, 0, map[string]any{"decision": decision})
	}
	fmt.Fprintf(stdout, "recorded reversible identity %s decision for %s\n", decision.Action, decision.ClusterID)
	return 0
}

func identityDecisionLogPath(projectPath string) string {
	return filepath.Join(projectPath, "data", "identity-decisions.jsonl")
}

func papersByCSVIndexes(papers []library.PaperRecord, csv string) ([]library.PaperRecord, bool) {
	indexes := splitCSV(csv)
	if len(indexes) == 0 {
		return nil, false
	}
	out := make([]library.PaperRecord, 0, len(indexes))
	for _, value := range indexes {
		index, err := strconv.Atoi(value)
		if err != nil || index < 0 || index >= len(papers) {
			return nil, false
		}
		out = append(out, papers[index])
	}
	return out, true
}

func refreshCrossrefDOIs(store library.Store, dois []string) (crossrefRefreshResult, error) {
	papers, err := store.List()
	if err != nil {
		return crossrefRefreshResult{code: "library_list_failed", exitCode: 1}, fmt.Errorf("list library: %v", err)
	}
	baseURL := os.Getenv("RFORGE_CROSSREF_URL")
	if baseURL == "" {
		baseURL = "https://api.crossref.org"
	}
	connector := sources.NewCrossrefConnector(defaultSourceHTTPClient(baseURL))
	updated := papers
	result := crossrefRefreshResult{}
	for _, doi := range dois {
		record, rawRef, err := connector.LookupDOI(context.Background(), doi)
		if err != nil {
			return crossrefRefreshResult{Refreshed: result.Refreshed, code: "library_refresh_failed", exitCode: 1}, fmt.Errorf("refresh DOI: %v", err)
		}
		refreshed, err := sources.PaperRecords(sources.SourceResponse{Records: []sources.SourceRecord{record}, RawRef: rawRef})
		if err != nil {
			return crossrefRefreshResult{Refreshed: result.Refreshed, code: "library_refresh_normalize_failed", exitCode: 1}, err
		}
		var ok bool
		updated, ok = refreshLibraryPaperByDOI(updated, refreshed[0])
		if ok {
			result.Refreshed++
		}
	}
	if err := store.ReplaceAll(updated); err != nil {
		return crossrefRefreshResult{Refreshed: result.Refreshed, code: "library_refresh_store_failed", exitCode: 1}, err
	}
	return result, nil
}

func refreshLibraryPaperByDOI(papers []library.PaperRecord, refreshed library.PaperRecord) ([]library.PaperRecord, bool) {
	for i, paper := range papers {
		if paper.Identifiers.DOI == refreshed.Identifiers.DOI {
			papers[i] = library.MergeDuplicate(paper, refreshed)
			if refreshed.Title != "" {
				papers[i].Title = refreshed.Title
			}
			if refreshed.Abstract != "" {
				papers[i].Abstract = refreshed.Abstract
			}
			if refreshed.Year != 0 {
				papers[i].Year = refreshed.Year
			}
			if refreshed.Venue != "" {
				papers[i].Venue = refreshed.Venue
			}
			if refreshed.Publisher != "" {
				papers[i].Publisher = refreshed.Publisher
			}
			if refreshed.License != "" {
				papers[i].License = refreshed.License
			}
			if refreshed.OpenAccess {
				papers[i].OpenAccess = true
			}
			if len(refreshed.URLs) > 0 {
				papers[i].URLs = refreshed.URLs
			}
			return papers, true
		}
	}
	return papers, false
}
